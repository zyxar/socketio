package engine

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func ExampleDial() {
	c, err := Dial("ws://localhost:8080/engine.io/", nil, WebsocketTransport)
	if err != nil {
		fmt.Printf("dial err=%s", err)
		return
	}
	defer c.Close()
	fmt.Printf("id=%s\n", c.Id())
}

func ExampleServer() {
	server, _ := NewServer(time.Second*5, time.Second*5, func(so *Socket) {
		so.On(EventMessage, Callback(func(typ MessageType, data []byte) {
			switch typ {
			case MessageTypeString:
				fmt.Fprintf(os.Stderr, "txt: %s\n", data)
			case MessageTypeBinary:
				fmt.Fprintf(os.Stderr, "bin: %x\n", data)
			default:
				fmt.Fprintf(os.Stderr, "???: %x\n", data)
			}
		}))
		so.On(EventPing, Callback(func(_ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "recv ping\n")
		}))
		so.On(EventClose, Callback(func(_ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket close\n")
		}))
		so.On(EventUpgrade, Callback(func(_ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket upgrade\n")
		}))
	})
	http.ListenAndServe(":8081", server)
}
