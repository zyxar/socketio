package socketio_test

import (
	"net/http"
	"time"

	"github.com/zyxar/socketio"
)

func ExampleServer() {
	server, _ := socketio.NewServer(time.Second*5, time.Second*5, socketio.DefaultParser)
	server.OnConnect(func(so *socketio.Socket) error {
		so.OnEvent("message", func(data interface{}) {

		})
		return so.Emit("event", "hello world!")
	})
	http.ListenAndServe(":8081", server)
	// Output:
}
