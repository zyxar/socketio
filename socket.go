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
	GetHeader(key string) string
	SetHeader(key, value string)
	Sid() string
	io.Closer
}

type nspSock struct {
	*socket
	name string
}

// Namespace implements Socket.Namespace
func (n *nspSock) Namespace() string { return n.name }

// Emit implements Socket.Emit
func (n *nspSock) Emit(event string, args ...interface{}) (err error) {
	return n.socket.emit(n.name, event, args...)
}

// EmitError implements Socket.EmitError
func (n *nspSock) EmitError(arg interface{}) (err error) {
	return n.socket.emitError(n.name, arg)
}

type socket struct {
	ß       *engine.Socket
	encoder Encoder
	decoder Decoder
	acks    map[string]*ackHandle
	mutex   sync.RWMutex
}

func newSocket(ß *engine.Socket, parser Parser) *socket {
	return &socket{
		ß:       ß,
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
}

type nspStore interface {
	getnsp(nsp string) (n *namespace, ok bool)
}

func detachall(s nspStore, sock *socket) {
	sock.mutex.Lock()
	for k := range sock.acks {
		delete(sock.acks, k)
		if nsp, ok := s.getnsp(k); ok {
			if nsp.onDisconnect != nil {
				nsp.onDisconnect(&nspSock{socket: sock, name: k})
			}
		}
	}
	sock.mutex.Unlock()
}

func (s *socket) fireAck(nsp string, id uint64, data []byte, buffer [][]byte, au ArgsUnmarshaler) (err error) {
	s.mutex.RLock()
	ack, ok := s.acks[nsp]
	s.mutex.RUnlock()
	if ok {
		err = ack.fireAck(&nspSock{socket: s, name: nsp}, id, data, buffer, au)
	}
	return
}

// Emit implements Socket.Emit
func (s *socket) Emit(event string, args ...interface{}) (err error) {
	return s.emit("/", event, args...)
}

// EmitError implements Socket.EmitError
func (s *socket) EmitError(arg interface{}) (err error) { return s.emitError("/", arg) }

// Namespace implements Socket.Namespace
func (*socket) Namespace() string { return "/" }

func (s *socket) emit(nsp string, event string, args ...interface{}) (err error) {
	s.mutex.RLock()
	ack, ok := s.acks[nsp]
	s.mutex.RUnlock()
	if !ok {
		return ErrorNamespaceUnavaialble
	}
	data := []interface{}{event}
	p := &Packet{Type: PacketTypeEvent, Namespace: nsp}
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
	if err = s.ß.Emit(engine.EventMessage, MessageTypeString, b); err != nil {
		return
	}
	for _, d := range bin {
		if err = s.ß.Emit(engine.EventMessage, MessageTypeBinary, d); err != nil {
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

func (s *socket) Sid() string                 { return s.ß.Sid() }
func (s *socket) Close() (err error)          { return s.ß.Close() }
func (s *socket) LocalAddr() net.Addr         { return s.ß.LocalAddr() }
func (s *socket) RemoteAddr() net.Addr        { return s.ß.RemoteAddr() }
func (s *socket) GetHeader(key string) string { return s.ß.GetHeader(key) }
func (s *socket) SetHeader(key, value string) { s.ß.SetHeader(key, value) }
