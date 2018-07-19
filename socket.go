package socketio

import (
	"errors"
	"io"
	"net"
	"reflect"
	"sync"

	"github.com/zyxar/socketio/engine"
)

var (
	// ErrorNamespaceUnavaialble indicates error of client accessing to a non-existent namespace
	ErrorNamespaceUnavaialble = errors.New("namespace unavailable")
)

// Socket is abstraction of bidirectional socket.io connection
type Socket interface {
	Emit(nsp string, event string, args ...interface{}) (err error)
	EmitError(nsp string, arg interface{}) (err error)
	On(nsp string, event string, callback interface{})
	OnDisconnect(fn func(nsp string))
	OnError(fn func(nsp string, err interface{}))
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	io.Closer
}

type socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	onError      func(nsp string, err interface{})
	onDisconnect func(nsp string)

	nsp     map[string]*nspHandle
	nspAttr map[string]struct{}
	mutex   sync.RWMutex
}

func newServerSocket(so *engine.Socket, parser Parser) *socket {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	socket := &socket{
		so:      so,
		encoder: encoder,
		decoder: decoder,
		nsp:     make(map[string]*nspHandle),
		nspAttr: make(map[string]struct{}),
	}
	socket.creatensp("/")
	socket.attachnsp("/")
	return socket
}

func newClientSocket(so *engine.Socket, parser Parser) *socket {
	return &socket{
		so:      so,
		encoder: parser.Encoder(),
		decoder: parser.Decoder(),
		nsp:     make(map[string]*nspHandle),
	}
}

func (s *socket) attachnsp(nsp string) *nspHandle {
	n, ok := s.namespace(nsp)
	if !ok {
		return nil
	}
	s.mutex.RLock()
	_, ok = s.nspAttr[nsp]
	s.mutex.RUnlock()
	if !ok {
		if err := s.emitPacket(&Packet{
			Type:      PacketTypeConnect,
			Namespace: nsp,
		}); err != nil {
			if s.onError != nil {
				s.onError(nsp, err)
			}
		}
		s.mutex.Lock()
		s.nspAttr[nsp] = struct{}{}
		s.mutex.Unlock()
	}
	return n
}

func (s *socket) detachnsp(nsp string) {
	s.mutex.Lock()
	delete(s.nspAttr, nsp)
	s.mutex.Unlock()
}

func (s *socket) creatensp(nsp string) *nspHandle {
	s.mutex.Lock()
	n, ok := s.nsp[nsp]
	if !ok {
		n = newNspHandle(nsp)
		s.nsp[nsp] = n
	}
	s.mutex.Unlock()
	return n
}

func (s *socket) namespace(nsp string) (*nspHandle, bool) {
	s.mutex.RLock()
	n, ok := s.nsp[nsp]
	s.mutex.RUnlock()
	return n, ok
}

func (s *socket) Emit(nsp string, event string, args ...interface{}) (err error) {
	namespace, ok := s.namespace(nsp)
	if !ok {
		return ErrorNamespaceUnavaialble
	}
	data := []interface{}{event}
	p := &Packet{
		Type:      PacketTypeEvent,
		Namespace: nsp,
	}
	for i := range args {
		if t := reflect.TypeOf(args[i]); t.Kind() == reflect.Func {
			p.ID = newid(namespace.store(args[i]))
		} else {
			data = append(data, args[i])
		}
	}
	p.Data = data
	return s.emitPacket(p)
}

func (s *socket) EmitError(nsp string, arg interface{}) (err error) {
	p := &Packet{
		Type:      PacketTypeError,
		Namespace: nsp,
		Data:      arg,
	}
	return s.emitPacket(p)
}

func (s *socket) ack(p *Packet) (err error) {
	p.Type = PacketTypeAck
	return s.emitPacket(p)
}

func (s *socket) emitPacket(p *Packet) (err error) {
	b, bin, err := s.encoder.Encode(p)
	if err != nil {
		return
	}
	if err = s.so.Emit(engine.EventMessage, MessageTypeString, b); err != nil {
		return
	}
	for _, d := range bin {
		if err = s.so.Emit(engine.EventMessage, MessageTypeBinary, d); err != nil {
			return
		}
	}
	return
}

func (s *socket) On(nsp string, event string, callback interface{}) {
	s.creatensp(nsp).On(event, callback)
}

func (s *socket) yield() *Packet {
	select {
	case p := <-s.decoder.Decoded():
		return p
	default:
		return nil
	}
}

func (s *socket) Close() (err error)                           { return s.so.Close() }
func (s *socket) OnError(fn func(nsp string, err interface{})) { s.onError = fn }
func (s *socket) OnDisconnect(fn func(nsp string))             { s.onDisconnect = fn }
func (s *socket) LocalAddr() net.Addr                          { return s.so.LocalAddr() }
func (s *socket) RemoteAddr() net.Addr                         { return s.so.RemoteAddr() }
