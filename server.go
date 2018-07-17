package socketio

import (
	"log"
	"net/http"
	"time"

	"github.com/zyxar/socketio/engine"
)

type Server struct {
	engine    *engine.Server
	onConnect func(so Socket) error
	onError   func(so Socket, err error)
}

func NewServer(interval, timeout time.Duration, parser Parser) (server *Server, err error) {
	e, err := engine.NewServer(interval, timeout, func(so *engine.Socket) {
		log.Println("socket open")

		socket, err := newServerSocket(so, parser)
		if err != nil {
			log.Println("new socket:", err)
			return
		}

		if server.onConnect != nil {
			if err = server.onConnect(socket); err != nil {
				if socket.onError != nil {
					socket.onError(err)
				}
			}
		}

		so.On(engine.EventMessage, engine.Callback(func(msgType engine.MessageType, data []byte) {
			switch msgType {
			case engine.MessageTypeString:
			case engine.MessageTypeBinary:
			default:
				return
			}
			if err := socket.decoder.Add(msgType, data); err != nil {
				if socket.onError != nil {
					socket.onError(err)
				}
			}
			if p := socket.yield(); p != nil {
				socket.process(p)
			}
		}))

		so.On(engine.EventClose, engine.Callback(func(_ engine.MessageType, _ []byte) {
			log.Println("socket close")
			socket.Close()
		}))
	})
	if err != nil {
		return
	}
	server = &Server{engine: e}
	return
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

func (s *Server) Close() error {
	return s.engine.Close()
}

func (s *Server) OnConnect(fn func(so Socket) error) {
	s.onConnect = fn
}

func (s *Server) OnError(fn func(so Socket, err error)) {
	s.onError = fn
}
