package socketio

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio/engine"
)

type Server struct {
	engine    *engine.Server
	onConnect func(so *Socket) error
	onError   func(so *Socket, err error)
}

func NewServer(interval, timeout time.Duration, parser Parser) (server *Server, err error) {
	e, err := engine.NewServer(interval, timeout)
	if err != nil {
		return
	}
	server = &Server{engine: e}

	e.On(engine.EventOpen, engine.Callback(func(so *engine.Socket, _ engine.MessageType, _ []byte) {
		log.Println("socket open")

		socket, err := newSocket(so, parser)
		if err != nil {
			log.Println("new socket:", err)
			return
		}

		if server.onConnect != nil {
			if err = server.onConnect(socket); err != nil {
				log.Println("on connect:", err)
			}
		}

		so.On(engine.EventMessage, engine.Callback(func(_ *engine.Socket, msgType engine.MessageType, data []byte) {
			switch msgType {
			case engine.MessageTypeString:
				log.Printf("txt: %s\n", data)
			case engine.MessageTypeBinary:
				log.Printf("bin: %x\n", data)
			default:
				log.Printf("???: %x\n", data)
				return
			}
			p, err := parser.Decode(data)
			if err != nil {
				log.Println("parser:", err.Error())
				return
			}
			switch p.Type {
			case PacketTypeConnect:
			case PacketTypeDisconnect:
				socket.Close()
			case PacketTypeEvent:
				if d, ok := p.Data.([]interface{}); ok {
					if event, ok := d[0].(string); ok {
						if len(d) > 1 {
							socket.fire(event, d[1:]...)
						} else {
							socket.fire(event)
						}
					}
				}
			case PacketTypeAck:
			case PacketTypeError:
			case PacketTypeBinaryEvent:
			case PacketTypeBinaryAck:
			default:
				log.Println("packet: ", p.Type)
			}
		}))

		so.On(engine.EventClose, engine.Callback(func(_ *engine.Socket, _ engine.MessageType, _ []byte) {
			log.Println("socket close")
		}))
	}))

	return
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	return s.engine.Close()
}

func (s *Server) OnConnect(fn func(so *Socket) error) {
	s.onConnect = fn
}

func (s *Server) OnError(fn func(so *Socket, err error)) {
	s.onError = fn
}
