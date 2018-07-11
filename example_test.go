package socketio_test

import (
	"net/http"
	"time"

	"github.com/zyxar/socketio"
)

func ExampleServer() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so *socketio.Socket) error {
		so.On("message", func(data string) {
			so.Emit("ack", "woot", func(msg string) {
			})
		})
		so.On("foobar", func(data string) (string, string) {
			return "foo", "bar"
		})
		go func() {
			for {
				select {
				case <-time.After(time.Second * 2):
					so.Emit("event", "check it out!")
				}
			}
		}()
		return so.Emit("event", "hello world!")
	})
	http.ListenAndServe(":8081", server)
}
