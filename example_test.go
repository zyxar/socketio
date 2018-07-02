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
	server.On(EventOpen, Handle(func(so *Socket, _ []byte) {
		so.On(EventMessage, Handle(func(_ *Socket, data []byte) {
			fmt.Fprintf(os.Stderr, "%x\n", data)
		}))
	}))
	http.ListenAndServe(":8081", server)
}
