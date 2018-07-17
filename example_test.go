package socketio_test

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio"
	"github.com/zyxar/socketio/engine"
)

func ExampleDial() {
	c, err := socketio.Dial("ws://localhost:8081/socket.io/", nil, engine.WebsocketTransport, socketio.DefaultParser)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer c.Close()
	c.On("/", "event", func(message string, b socketio.Bytes) {
		log.Printf("%s => %x", message, b.Marshal())
	})
	c.OnError(func(err error) {
		log.Println(err.Error())
	})
	c.Emit("/", "binary", "bytes", &socketio.Bytes{[]byte{1, 2, 3, 4, 5, 6}})

	for {
		select {
		case <-time.After(time.Second * 2):
		}
		c.Emit("/", "foobar", "foo", func(a, b string) {
			log.Println("foobar =>", a, b)
		})
	}
}

func ExampleServer() {
	server := newServer()
	defer server.Close()
	http.ListenAndServe("localhost:8081", server)
}

func newServer() *socketio.Server {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so socketio.Socket) error {
		so.On("/", "message", func(data string) {
			so.Emit("/", "ack", "woot", func(msg string, b *socketio.Bytes) {
				log.Printf("%s=> %x", msg, b.Marshal())
			})
		})
		so.On("/", "binary", func(data interface{}, b socketio.Bytes) {
			log.Printf("%s <- %x", data, b.Marshal())
		})
		so.On("/ditto", "disguise", func(msg interface{}, b socketio.Bytes) {
			log.Printf("%v: %x", msg, b.Marshal())
		})
		so.On("/", "foobar", func(data string) (string, string) {
			log.Println("foobar:", data)
			return "foo", "bar"
		})
		so.OnError(func(err error) {
			log.Println("socket error:", err)
		})
		go func() {
			b := &socketio.Bytes{}
			for {
				select {
				case <-time.After(time.Second * 2):
					t, _ := time.Now().MarshalBinary()
					b.Unmarshal(t)
					if err := so.Emit("/", "event", "check it out!", b); err != nil {
						log.Println(err)
						return
					}
				}
			}
		}()
		return so.Emit("/", "event", "hello world!")
	})
	return server
}
