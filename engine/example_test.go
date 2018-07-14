package engine_test

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zyxar/socketio/engine"
)

func ExampleDial() {
	c, err := engine.Dial("ws://localhost:8080/engine.io/", nil, engine.WebsocketTransport)
	if err != nil {
		fmt.Printf("dial err=%s", err)
		return
	}
	defer c.Close()
	fmt.Printf("id=%s\n", c.Id())
}

func ExampleServer() {
	server := newServer()
	defer server.Close()
	http.ListenAndServe(":8081", server)
}

func newServer() *engine.Server {
	server, _ := engine.NewServer(time.Second*5, time.Second*5, func(so *engine.Socket) {
		so.On(engine.EventMessage, engine.Callback(func(typ engine.MessageType, data []byte) {
			switch typ {
			case engine.MessageTypeString:
				fmt.Fprintf(os.Stderr, "txt: %s\n", data)
			case engine.MessageTypeBinary:
				fmt.Fprintf(os.Stderr, "bin: %x\n", data)
			default:
				fmt.Fprintf(os.Stderr, "???: %x\n", data)
			}
		}))
		so.On(engine.EventPing, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket ping\n")
		}))
		so.On(engine.EventClose, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket close\n")
		}))
		so.On(engine.EventUpgrade, engine.Callback(func(_ engine.MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket upgrade\n")
		}))
	})
	return server
}
