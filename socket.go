package socketio

import (
	"io"
	"net"
	"reflect"
	"sync"

	"github.com/zyxar/socketio/engine"
)

type Socket interface {
	Emit(nsp string, event string, args ...interface{}) (err error)
	On(nsp string, event string, callback interface{})
	OnError(fn func(nsp string, err interface{}))
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	io.Closer
}

type socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	onError func(nsp string, err interface{})

	nsp   map[string]*nspHandle
	nspL  map[string]struct{}
	mutex sync.RWMutex
}

func newServerSocket(so *engine.Socket, parser Parser) *socket {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	socket := &socket{
		so:      so,
		encoder: encoder,
		decoder: decoder,
		nsp:     make(map[string]*nspHandle),
		nspL:    make(map[string]struct{}),
	}

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
	if nsp == "" {
		nsp = "/"
	}
	s.mutex.RLock()
	_, ok := s.nspL[nsp]
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
		s.nspL[nsp] = struct{}{}
		s.mutex.Unlock()
	}
	return s.namespace(nsp)
}

func (s *socket) detachnsp(nsp string) {
	s.mutex.Lock()
	delete(s.nspL, nsp)
	s.mutex.Unlock()
}

func (s *socket) namespace(nsp string) *nspHandle {
	if nsp == "" {
		nsp = "/"
	}
	s.mutex.RLock()
	n, ok := s.nsp[nsp]
	s.mutex.RUnlock()
	if !ok {
		n = newNspHandle(nsp)
		s.mutex.Lock()
		s.nsp[nsp] = n
		s.mutex.Unlock()
	}
	return n
}

func (s *socket) Emit(nsp string, event string, args ...interface{}) (err error) {
	data := []interface{}{event}
	p := &Packet{
		Type:      PacketTypeEvent,
		Namespace: nsp,
	}
	for i := range args {
		if t := reflect.TypeOf(args[i]); t.Kind() == reflect.Func {
			p.ID = newid(s.namespace(nsp).store(args[i]))
		} else {
			data = append(data, args[i])
		}
	}
	p.Data = data
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
	s.namespace(nsp).On(event, callback)
}

func (s *socket) yield() *Packet {
	select {
	case p := <-s.decoder.Decoded():
		return p
	default:
		return nil
	}
}

func (s *socket) Close() (err error) {
	return s.so.Close()
}

func (s *socket) OnError(fn func(nsp string, err interface{})) {
	s.onError = fn
}

func (s *socket) LocalAddr() net.Addr  { return s.so.LocalAddr() }
func (s *socket) RemoteAddr() net.Addr { return s.so.RemoteAddr() }
