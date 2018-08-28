package socketio

import (
	"log"
	"net/http"
	"time"

	"github.com/tinylib/msgp/msgp"
)

func ExampleDial() {
	c := NewClient()
	c.Namespace("/").
		OnConnect(func(so Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
			if err := so.Emit("binary", "bytes", &Bytes{Data: []byte{1, 2, 3, 4, 5, 6}}); err != nil {
				log.Println("so.Emit:", err)
			}
		}).
		OnDisconnect(func(so Socket) {
			log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
		}).
		OnError(func(so Socket, err ...interface{}) {
			log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
		}).
		OnEvent("event", func(message string, b Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s => %x", message, bb)
		})

	err := c.Dial("ws://localhost:8081/socket.io/", nil, WebsocketTransport, DefaultParser)
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
	server, _ := NewServer(time.Second*5, time.Second*5, DefaultParser)
	var onConnect = func(so Socket) {
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

	var onDisconnect = func(so Socket) {
		log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
	}

	var onError = func(so Socket, err ...interface{}) {
		log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
	}

	server.Namespace("/").
		OnConnect(onConnect).
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("message", func(so Socket, data string) {
			if err := so.Emit("ack", "woot", func(msg string, b *Bytes) {
				bb, _ := b.MarshalBinary()
				log.Printf("%s=> %x", msg, bb)
			}); err != nil {
				log.Println("emit:", err)
			}
		}).
		OnEvent("binary", func(data interface{}, b Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s <- %x", data, bb)
		}).
		OnEvent("foobar", func(data string) (string, string) {
			log.Println("foobar:", data)
			return "foo", "bar"
		})

	server.Namespace("/ditto").
		OnConnect(func(so Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
		}).
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("disguise", func(msg interface{}, b Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%v: %x", msg, bb)
		})

	server.OnError(func(err error) {
		log.Printf("server error: %v", err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}

func ExampleServer_withMsgpackParser() {
	server, _ := NewServer(time.Second*5, time.Second*5, MsgpackParser)
	server.Namespace("/").
		OnConnect(func(so Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
			so.Emit("event", "hello world!", time.Now())
		}).
		OnDisconnect(func(so Socket) {
			log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
		}).
		OnEvent("message", func(b msgp.Raw, data foobar) {
			log.Printf("%x %v", b, data)
		}).
		OnError(func(so Socket, err ...interface{}) {
			log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
		})
	server.OnError(func(err error) {
		log.Println("server err:", err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}
