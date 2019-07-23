package socketio

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/zyxar/socketio/engine"
)

// Server is socket.io server implementation
type Server struct {
	engine   *engine.Server
	sockets  map[*engine.Socket]*socket
	sockLock sync.RWMutex
	onError  func(err error)
	nsps     map[string]*namespace
}

// NewServer creates a socket.io server instance upon underlying engine.io transport
func NewServer(ctx context.Context, interval, timeout time.Duration, parser Parser) (server *Server, err error) {
	e, err := engine.NewServer(ctx, interval, timeout, func(ß *engine.Socket) {
		socket := newSocket(ß, parser)
		socket.attachnsp("/")
		nsp := server.creatensp("/")
		if err := socket.emitPacket(&Packet{
			Type:      PacketTypeConnect,
			Namespace: "/",
		}); err != nil {
			if nsp.onError != nil {
				nsp.onError(socket, err)
			}
		}
		server.sockLock.Lock()
		server.sockets[ß] = socket
		server.sockLock.Unlock()
		if nsp.onConnect != nil {
			nsp.onConnect(socket)
		}
	})
	if err != nil {
		return
	}
	server = &Server{engine: e, sockets: make(map[*engine.Socket]*socket), nsps: make(map[string]*namespace)}

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
				server.onError(err)
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
		detachall(server, socket)
	}))

	return
}

// Namespace ensures a Namespace instance exists in server
func (s *Server) Namespace(nsp string) Namespace { return s.creatensp(nsp) }

func (s *Server) creatensp(nsp string) *namespace {
	n, ok := s.nsps[nsp]
	if !ok {
		n = &namespace{callbacks: make(map[string]*callback)}
		s.nsps[nsp] = n
	}
	return n
}

func (s *Server) getnsp(nsp string) (n *namespace, ok bool) { n, ok = s.nsps[nsp]; return }

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.engine.ServeHTTP(w, r) }

// Close closes underlying engine.io transport
func (s *Server) Close() error { return s.engine.Close() }

// OnError registers fn as callback for error handling
func (s *Server) OnError(fn func(err error)) { s.onError = fn }

// process is the Packet process handle on server side
func (s *Server) process(sock *socket, p *Packet) {
	nsp, ok := s.getnsp(p.Namespace)
	if !ok {
		if p.Type > PacketTypeDisconnect {
			sock.emitError(p.Namespace, ErrorNamespaceUnavaialble.Error())
		}
		return
	}
	switch p.Type {
	case PacketTypeConnect:
		sock.attachnsp(p.Namespace)
		if err := sock.emitPacket(&Packet{
			Type:      PacketTypeConnect,
			Namespace: p.Namespace,
		}); err != nil {
			if nsp.onError != nil {
				nsp.onError(&nspSock{socket: sock, name: p.Namespace}, err)
			}
		}
		if nsp.onConnect != nil {
			nsp.onConnect(&nspSock{socket: sock, name: p.Namespace})
		}
	case PacketTypeDisconnect:
		sock.detachnsp(p.Namespace)
		if nsp.onDisconnect != nil {
			nsp.onDisconnect(&nspSock{socket: sock, name: p.Namespace})
		}
	case PacketTypeEvent, PacketTypeBinaryEvent:
		event, data, bin, err := sock.decoder.ParseData(p)
		if err != nil {
			if nsp.onError != nil {
				nsp.onError(&nspSock{socket: sock, name: p.Namespace}, err)
			}
			return
		}
		if event == "" {
			return
		}
		v, err := nsp.fireEvent(&nspSock{socket: sock, name: p.Namespace}, event, data, bin, sock.decoder)
		if err != nil {
			if nsp.onError != nil {
				nsp.onError(&nspSock{socket: sock, name: p.Namespace}, err)
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
				if nsp.onError != nil {
					nsp.onError(&nspSock{socket: sock, name: p.Namespace}, err)
				}
			}
		}
	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil {
			_, data, bin, err := sock.decoder.ParseData(p)
			if err != nil {
				if nsp.onError != nil {
					nsp.onError(&nspSock{socket: sock, name: p.Namespace}, err)
				}
				return
			}
			sock.fireAck(p.Namespace, *p.ID, data, bin, sock.decoder)
		}
	case PacketTypeError:
		if nsp.onError != nil {
			nsp.onError(&nspSock{socket: sock, name: p.Namespace}, p.Data)
		}
	default:
		if nsp.onError != nil {
			nsp.onError(&nspSock{socket: sock, name: p.Namespace}, ErrUnknownPacket)
		}
	}
}

var (
	WebsocketTransport = engine.WebsocketTransport
)
