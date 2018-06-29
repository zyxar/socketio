package engineio

import (
	"fmt"
	"log"
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
