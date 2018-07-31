package socketio_test

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio"
	"github.com/zyxar/socketio/engine"
)

func ExampleDial() {
	c, err := socketio.Dial("ws://localhost:8081/socket.io/", nil, engine.WebsocketTransport, socketio.DefaultParser,
		func(so socketio.Socket) {
			log.Println("connected:", so.RemoteAddr(), so.Sid(), so.Namespace())
			if err := so.Emit("binary", "bytes", &socketio.Bytes{Data: []byte{1, 2, 3, 4, 5, 6}}); err != nil {
				log.Println("Emit:", err)
			}
		})
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer c.Close()
	var onDisconnect = func(so socketio.Socket) {
		log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
	}

	var onError = func(so socketio.Socket, err ...interface{}) {
		log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
	}
	c.Namespace("/").
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("event", func(message string, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s => %x", message, bb)
		})

	for {
		<-time.After(time.Second * 2)
		if err := c.Emit("/", "foobar", "foo", func(a, b string) {
			log.Println("foobar =>", a, b)
		}); err != nil {
			log.Println(err)
			break
		}
	}
}

func ExampleServer() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so socketio.Socket) {
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
	})

	var onDisconnect = func(so socketio.Socket) {
		log.Printf("%v %v %q disconnected", so.Sid(), so.RemoteAddr(), so.Namespace())
	}

	var onError = func(so socketio.Socket, err ...interface{}) {
		log.Println("socket", so.Sid(), so.RemoteAddr(), so.Namespace(), "error:", err)
	}

	server.Namespace("/").
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
		OnDisconnect(onDisconnect).
		OnError(onError).
		OnEvent("disguise", func(msg interface{}, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%v: %x", msg, bb)
		})

	server.OnError(func(so socketio.Socket, err error) {
		log.Printf("%v error=> %v", so.RemoteAddr(), err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}
