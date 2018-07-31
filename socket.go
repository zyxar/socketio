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
	Emit(event string, args ...interface{}) (err error)
	EmitError(arg interface{}) (err error)
	Namespace() string
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	Sid() string
	io.Closer
	// OnDisconnect(fn func(string))
	// OnError(fn func(nsp string, err interface{}))
}

type nspSock struct {
	*socket
	name string
}

func (n *nspSock) Namespace() string { return n.name }

func (n *nspSock) Emit(event string, args ...interface{}) (err error) {
	return n.socket.emit(n.name, event, args...)
}

func (n *nspSock) EmitError(arg interface{}) (err error) {
	return n.socket.emitError(n.name, arg)
}

type socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	onError      func(nsp string, err interface{})
	onDisconnect func(nsp string)

	acks  map[string]*ackHandle
	mutex sync.RWMutex
}

func newServerSocket(so *engine.Socket, parser Parser) *socket {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	socket := &socket{
		so:      so,
		encoder: encoder,
		decoder: decoder,
		acks:    make(map[string]*ackHandle),
	}
	socket.attachnsp("/")
	return socket
}

func newClientSocket(so *engine.Socket, parser Parser) *socket {
	return &socket{
		so:      so,
		encoder: parser.Encoder(),
		decoder: parser.Decoder(),
		acks:    make(map[string]*ackHandle),
	}
}

func (s *socket) attachnsp(nsp string) {
	s.mutex.Lock()
	s.acks[nsp] = &ackHandle{ackmap: make(map[uint64]*callback)}
	s.mutex.Unlock()
}

func (s *socket) detachnsp(nsp string) {
	s.mutex.Lock()
	_, ok := s.acks[nsp]
	if ok {
		delete(s.acks, nsp)
	}
	s.mutex.Unlock()
	if ok && s.onDisconnect != nil {
		s.onDisconnect(nsp)
	}
}

func (s *socket) detachall() {
	s.mutex.Lock()
	for k := range s.acks {
		delete(s.acks, k)
		if s.onDisconnect != nil {
			s.onDisconnect(k)
		}
	}
	s.mutex.Unlock()
}

func (s *socket) fireAck(nsp string, id uint64, data []byte, buffer [][]byte, au ArgsUnmarshaler) (err error) {
	s.mutex.RLock()
	ack, ok := s.acks[nsp]
	s.mutex.RUnlock()
	if ok {
		err = ack.fireAck(&nspSock{s, nsp}, id, data, buffer, au)
	}
	return
}

func (s *socket) emit(nsp string, event string, args ...interface{}) (err error) {
	s.mutex.RLock()
	ack, ok := s.acks[nsp]
	s.mutex.RUnlock()
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
			p.ID = newid(ack.onAck(args[i]))
		} else {
			data = append(data, args[i])
		}
	}
	p.Data = data
	return s.emitPacket(p)
}

func (s *socket) emitError(nsp string, arg interface{}) (err error) {
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

func (s *socket) yield() *Packet {
	select {
	case p := <-s.decoder.Decoded():
		return p
	default:
		return nil
	}
}

func (s *socket) Sid() string                                  { return s.so.Sid() }
func (s *socket) Close() (err error)                           { return s.so.Close() }
func (s *socket) OnError(fn func(nsp string, err interface{})) { s.onError = fn }
func (s *socket) OnDisconnect(fn func(nsp string))             { s.onDisconnect = fn }
func (s *socket) LocalAddr() net.Addr                          { return s.so.LocalAddr() }
func (s *socket) RemoteAddr() net.Addr                         { return s.so.RemoteAddr() }
