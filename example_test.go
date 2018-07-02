package engio

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func ExampleDial() {
	c, err := Dial("ws://localhost:8080/engine.io/", nil, WebsocketTransport)
	if err != nil {
		log.Fatalf("dial err=%s", err)
		return
	}
	defer c.Close()
	log.Printf("id=%s\n", c.Id())
	fmt.Printf("interval=%s, timeout=%s\n", c.pingInterval, c.pingTimeout)
	// //Output: interval=25s, timeout=5s
}

func ExampleServer() {
	server, _ := NewServer()
	server.On(EventOpen, Callback(func(so *Socket, _ MessageType, _ []byte) {
		so.On(EventMessage, Callback(func(_ *Socket, typ MessageType, data []byte) {
			switch typ {
			case MessageTypeString:
				fmt.Fprintf(os.Stderr, "txt: %s\n", data)
			case MessageTypeBinary:
				fmt.Fprintf(os.Stderr, "bin: %x\n", data)
			default:
				fmt.Fprintf(os.Stderr, "???: %x\n", data)
			}
		}))
		so.On(EventPing, Callback(func(_ *Socket, _ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "recv ping\n")
		}))
		so.On(EventClose, Callback(func(_ *Socket, _ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket close\n")
		}))
		so.On(EventUpgrade, Callback(func(_ *Socket, _ MessageType, _ []byte) {
			fmt.Fprintf(os.Stderr, "socket upgrade\n")
		}))
	}))
	http.ListenAndServe(":8081", server)
}
