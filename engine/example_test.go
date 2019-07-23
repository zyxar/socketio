package engine

import (
	"context"
	"log"
	"net/http"
	"time"
)

func ExampleDial() {
	c, err := Dial(context.Background(), "ws://localhost:8080/engine.io/", nil, WebsocketTransport)
	if err != nil {
		log.Printf("dial err=%s", err)
		return
	}
	defer c.Close()
	log.Printf("id=%s\n", c.Sid())
}

func ExampleServer() {
	server, _ := NewServer(context.Background(), time.Second*5, time.Second*5, func(so *Socket) {
		log.Println("connect", so.RemoteAddr())
	})
	server.On(EventMessage, Callback(func(so *Socket, typ MessageType, data []byte) {
		switch typ {
		case MessageTypeString:
			log.Printf("txt: %s\n", data)
		case MessageTypeBinary:
			log.Printf("bin: %x\n", data)
		default:
			log.Printf("???: %x\n", data)
		}
	}))
	server.On(EventPing, Callback(func(so *Socket, _ MessageType, _ []byte) {
		log.Printf("socket ping\n")
	}))
	server.On(EventClose, Callback(func(so *Socket, _ MessageType, _ []byte) {
		log.Printf("socket close\n")
	}))
	server.On(EventUpgrade, Callback(func(so *Socket, _ MessageType, _ []byte) {
		log.Printf("socket upgrade\n")
	}))
	defer server.Close()
	log.Fatalln(http.ListenAndServe("localhost:8081", server))
}
