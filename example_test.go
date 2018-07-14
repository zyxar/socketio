package socketio_test

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio"
)

func ExampleServer() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so *socketio.Socket) error {
		so.On("message", func(data string) {
			so.Emit("ack", "woot", func(msg string, b *socketio.Binary) {
				log.Printf("%s=> %x", msg, b.Bytes())
			})
		})
		so.On("binary", func(data interface{}, b *socketio.Binary) {
			log.Println(data)
			log.Printf("%x", b.Bytes())
		})
		so.On("foobar", func(data string) (string, string) {
			log.Println("foobar:", data)
			return "foo", "bar"
		})
		so.OnError(func(err error) {
			log.Println("socket error:", err)
		})
		go func() {
			b := &socketio.Binary{}
			for {
				select {
				case <-time.After(time.Second * 2):
					t, _ := time.Now().MarshalBinary()
					b.Attach(t)
					so.Emit("event", "check it out!", b)
				}
			}
		}()
		return so.Emit("event", "hello world!")
	})
	http.ListenAndServe(":8081", server)
}
