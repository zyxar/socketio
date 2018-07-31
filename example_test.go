package socketio_test

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio"
	"github.com/zyxar/socketio/engine"

	"github.com/tinylib/msgp/msgp"
)

func ExampleDial() {
	c := socketio.NewClient()
	c.Namespace("/").
		OnConnect(func(so socketio.Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
			if err := so.Emit("binary", "bytes", &socketio.Bytes{Data: []byte{1, 2, 3, 4, 5, 6}}); err != nil {
				log.Println("so.Emit:", err)
			}
		}).
		OnDisconnect(func(so socketio.Socket) {
			log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
		}).
		OnError(func(so socketio.Socket, err ...interface{}) {
			log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
		}).
		OnEvent("event", func(message string, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s => %x", message, bb)
		})

	err := c.Dial("ws://localhost:8081/socket.io/", nil, engine.WebsocketTransport, socketio.DefaultParser)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer c.Close()

	for {
		<-time.After(time.Second * 2)
		if err := c.Emit("/", "foobar", "foo", func(a, b string) {
			log.Println("foobar =>", a, b)
		}); err != nil {
			log.Println("c.Emit:", err)
			break
		}
	}
}

func ExampleServer() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	var onConnect = func(so socketio.Socket) {
		log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
		go func() {
			for {
				<-time.After(time.Second * 2)
				if err := so.Emit("event", "check it out!", time.Now()); err != nil {
					log.Println("emit:", err)
					return
				}
			}
		}()
		so.Emit("event", "hello world!")
	}

	var onDisconnect = func(so socketio.Socket) {
		log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
	}

	var onError = func(so socketio.Socket, err ...interface{}) {
		log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
	}

	server.Namespace("/").
		OnConnect(onConnect).
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("message", func(so socketio.Socket, data string) {
			if err := so.Emit("ack", "woot", func(msg string, b *socketio.Bytes) {
				bb, _ := b.MarshalBinary()
				log.Printf("%s=> %x", msg, bb)
			}); err != nil {
				log.Println("emit:", err)
			}
		}).
		OnEvent("binary", func(data interface{}, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s <- %x", data, bb)
		}).
		OnEvent("foobar", func(data string) (string, string) {
			log.Println("foobar:", data)
			return "foo", "bar"
		})

	server.Namespace("/ditto").
		OnConnect(func(so socketio.Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
		}).
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("disguise", func(msg interface{}, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%v: %x", msg, bb)
		})

	server.OnError(func(err error) {
		log.Printf("server error: %v", err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}

func ExampleServer2() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.MsgpackParser)
	server.Namespace("/").
		OnConnect(func(so socketio.Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
			so.Emit("event", "hello world!", time.Now())
		}).
		OnDisconnect(func(so socketio.Socket) {
			log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
		}).
		OnEvent("message", func(b msgp.Raw, data foobar) {
			log.Printf("%x %v", b, data)
		}).
		OnError(func(so socketio.Socket, err ...interface{}) {
			log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
		})
	server.OnError(func(err error) {
		log.Println("server err:", err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}

type foobar struct {
	Foo string `msg:"foo"`
}

// MarshalMsg implements msgp.Marshaler
func (z foobar) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "foo"
	o = append(o, 0x81, 0xa3, 0x66, 0x6f, 0x6f)
	o = msgp.AppendString(o, z.Foo)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *foobar) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "foo":
			z.Foo, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z foobar) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Foo)
	return
}
