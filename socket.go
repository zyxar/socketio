package socketio

import (
	"reflect"
	"sync"

	"github.com/zyxar/socketio/engine"
)

type Socket interface {
	Emit(nsp string, event string, args ...interface{}) (err error)
	On(nsp string, event string, callback interface{})
	OnError(fn func(err error))
}

type socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	onError func(err error)
	nspCtor func(nsp string) *Namespace

	nsp   map[string]*Namespace
	mutex sync.RWMutex
}

func newServerSocket(so *engine.Socket, parser Parser) (*socket, error) {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	nspOnConnect := func(nsp string) error {
		b, _ := encoder.Encode(&Packet{
			Type:      PacketTypeConnect,
			Namespace: nsp,
		})
		return so.Emit(engine.EventMessage, MessageTypeString, b[0])
	}
	if err := nspOnConnect("/"); err != nil {
		return nil, err
	}
	socket := &socket{
		so:      so,
		encoder: encoder,
		decoder: decoder,
		nsp:     map[string]*Namespace{"/": newNamespace("/")},
		nspCtor: func(nsp string) *Namespace {
			nspOnConnect(nsp)
			return newNamespace(nsp)
		},
	}
	return socket, nil
}

func newClientSocket(so *engine.Socket, parser Parser) *socket {
	return &socket{
		so:      so,
		encoder: parser.Encoder(),
		decoder: parser.Decoder(),
		nsp:     make(map[string]*Namespace),
		nspCtor: newNamespace,
	}
}

func (s *socket) namespace(nsp string) *Namespace {
	if nsp == "" {
		nsp = "/"
	}
	s.mutex.RLock()
	n, ok := s.nsp[nsp]
	s.mutex.RUnlock()
	if !ok {
		n = s.nspCtor(nsp)
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
	b, _ := s.encoder.Encode(p)
	if err = s.so.Emit(engine.EventMessage, MessageTypeString, b[0]); err != nil {
		return
	}
	for _, d := range b[1:] {
		if err = s.so.Emit(engine.EventMessage, MessageTypeBinary, d); err != nil {
			return
		}
	}
	return
}

func (s *socket) ack(p *Packet) (err error) {
	p.Type = PacketTypeAck
	b, err := s.encoder.Encode(p)
	if err != nil {
		return
	}
	if err = s.so.Emit(engine.EventMessage, MessageTypeString, b[0]); err != nil {
		return
	}
	for _, d := range b[1:] {
		if err = s.so.Emit(engine.EventMessage, MessageTypeBinary, d); err != nil {
			return
		}
	}
	return
}

func (s *socket) On(nsp string, event string, callback interface{}) {
	s.namespace(nsp).On(event, callback)
}

func (s *socket) fire(nsp string, event string, args []byte, buffer [][]byte) ([]reflect.Value, error) {
	return s.namespace(nsp).fire(event, args, buffer)
}

func (s *socket) process(p *Packet) {
	switch p.Type {
	case PacketTypeConnect:
		s.fire(p.Namespace, "connect", nil, nil) // client
	case PacketTypeDisconnect:
		s.mutex.Lock()
		delete(s.nsp, p.Namespace)
		s.mutex.Unlock()
	case PacketTypeEvent, PacketTypeBinaryEvent:
		if p.event != nil {
			v, err := s.fire(p.Namespace, p.event.name, p.event.data, p.buffer)
			if err != nil {
				if s.onError != nil {
					s.onError(err)
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
				if err = s.ack(p); err != nil {
					if s.onError != nil {
						s.onError(err)
					}
				}
			}
		}
	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil && p.event != nil {
			s.namespace(p.Namespace).onAck(*p.ID, p.event.data, p.buffer)
		}
	case PacketTypeError:
	default:
		if s.onError != nil {
			s.onError(ErrUnknownPacket)
		}
	}
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

func (s *socket) OnError(fn func(err error)) {
	s.onError = fn
}
