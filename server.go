package socketio

import (
	"net/http"
	"sync"
	"time"

	"github.com/zyxar/socketio/engine"
)

// Server is socket.io server implementation
type Server struct {
	engine    *engine.Server
	sockets   map[*engine.Socket]*socket
	sockLock  sync.RWMutex
	onConnect func(so Socket) error
	onError   func(so Socket, err error)
}

// NewServer creates a socket.io server instance upon underlying engine.io transport
func NewServer(interval, timeout time.Duration, parser Parser) (server *Server, err error) {
	e, err := engine.NewServer(interval, timeout, func(ß *engine.Socket) {
		socket := newServerSocket(ß, parser)
		if server.onConnect != nil {
			if err = server.onConnect(socket); err != nil {
				if server.onError != nil {
					server.onError(socket, err)
				}
			}
		}
		if err := socket.emitPacket(&Packet{
			Type:      PacketTypeConnect,
			Namespace: "/",
		}); err != nil {
			if socket.onError != nil {
				socket.onError("/", err)
			}
		}
		server.sockLock.Lock()
		server.sockets[ß] = socket
		server.sockLock.Unlock()
	})
	if err != nil {
		return
	}
	server = &Server{engine: e, sockets: make(map[*engine.Socket]*socket)}

	e.On(engine.EventMessage, engine.Callback(func(ß *engine.Socket, msgType engine.MessageType, data []byte) {
		server.sockLock.RLock()
		socket := server.sockets[ß]
		server.sockLock.RUnlock()
		if socket == nil {
			return
		}
		switch msgType {
		case engine.MessageTypeString:
		case engine.MessageTypeBinary:
		default:
			return
		}
		if err := socket.decoder.Add(msgType, data); err != nil {
			if server.onError != nil {
				server.onError(socket, err)
			}
		}
		if p := socket.yield(); p != nil {
			server.process(socket, p)
		}
	}))

	e.On(engine.EventClose, engine.Callback(func(ß *engine.Socket, _ engine.MessageType, _ []byte) {
		server.sockLock.RLock()
		socket := server.sockets[ß]
		server.sockLock.RUnlock()
		if socket == nil {
			return
		}
		server.sockLock.Lock()
		delete(server.sockets, ß)
		server.sockLock.Unlock()
		socket.Close()
		socket.detachall()
	}))

	return
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.engine.ServeHTTP(w, r)
}

// Close closes underlying engine.io transport
func (s *Server) Close() error {
	return s.engine.Close()
}

// OnConnect registers fn as callback to be called when a socket connects
func (s *Server) OnConnect(fn func(so Socket) error) {
	s.onConnect = fn
}

// OnError registers fn as callback for error handling
func (s *Server) OnError(fn func(so Socket, err error)) {
	s.onError = fn
}

// process is the Packet process handle on server side
func (*Server) process(sock *socket, p *Packet) {
	nsp := sock.attachnsp(p.Namespace)
	if nsp == nil {
		if p.Type > PacketTypeDisconnect {
			sock.EmitError(p.Namespace, ErrorNamespaceUnavaialble.Error())
		}
		return
	}
	switch p.Type {
	case PacketTypeConnect:
		if err := sock.emitPacket(&Packet{
			Type:      PacketTypeConnect,
			Namespace: p.Namespace,
		}); err != nil {
			if sock.onError != nil {
				sock.onError(p.Namespace, err)
			}
		}
	case PacketTypeDisconnect:
		sock.detachnsp(p.Namespace)
	case PacketTypeEvent, PacketTypeBinaryEvent:
		event, data, bin, err := sock.decoder.ParseData(p)
		if err != nil {
			if sock.onError != nil {
				sock.onError(p.Namespace, err)
			}
			return
		}
		if event == "" {
			return
		}
		v, err := nsp.fireEvent(event, data, bin, sock.decoder)
		if err != nil {
			if sock.onError != nil {
				sock.onError(p.Namespace, err)
			}
			return
		}
		if p.ID != nil {
			p.Data = nil
			if v != nil {
				d := make([]interface{}, len(v))
				for i := range d {
					d[i] = v[i].Interface()
				}
				p.Data = d
			}
			if err = sock.ack(p); err != nil {
				if sock.onError != nil {
					sock.onError(p.Namespace, err)
				}
			}
		}

	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil {
			_, data, bin, err := sock.decoder.ParseData(p)
			if err != nil {
				if sock.onError != nil {
					sock.onError(p.Namespace, err)
				}
				return
			}
			nsp.fireAck(*p.ID, data, bin, sock.decoder)
		}
	case PacketTypeError:
		if sock.onError != nil {
			sock.onError(p.Namespace, p.Data)
		}
	default:
		if sock.onError != nil {
			sock.onError(p.Namespace, ErrUnknownPacket)
		}
	}
}
