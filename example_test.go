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
		})
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer c.Close()
	c.Namespace("/").
		OnEvent("event", func(message string, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%s => %x", message, bb)
		})
	c.OnError(func(err interface{}) {
		log.Println(err)
	})
	if err = c.Emit("/", "binary", "bytes", &socketio.Bytes{Data: []byte{1, 2, 3, 4, 5, 6}}); err != nil {
		log.Println(err)
	}

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
					log.Println(err)
					return
				}
			}
		}()
		so.Emit("event", "hello world!")
	})

	server.Namespace("/").
		OnEvent("message", func(so socketio.Socket, data string) {
			if err := so.Emit("ack", "woot", func(msg string, b *socketio.Bytes) {
				bb, _ := b.MarshalBinary()
				log.Printf("%s=> %x", msg, bb)
			}); err != nil {
				log.Println(err)
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
		OnEvent("disguise", func(msg interface{}, b socketio.Bytes) {
			bb, _ := b.MarshalBinary()
			log.Printf("%v: %x", msg, bb)
		})

	// so.OnDisconnect(func(nsp string) {
	// 	log.Printf("%q disconnected", nsp)
	// })
	// so.OnError(func(nsp string, err interface{}) {
	// 	log.Printf("socket nsp=%q, error=%v", nsp, err)
	// })

	server.OnError(func(so socketio.Socket, err error) {
		log.Printf("%v error=> %v", so.RemoteAddr(), err)
	})
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}
