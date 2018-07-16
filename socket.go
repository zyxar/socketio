package socketio

import (
	"reflect"
	"sync"

	"github.com/zyxar/socketio/engine"
)

type Socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	onError func(err error)

	nsp   map[string]*Namespace
	mutex sync.RWMutex
}

func newSocket(so *engine.Socket, parser Parser) (*Socket, error) {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	b, _ := encoder.Encode(&Packet{
		Type:      PacketTypeConnect,
		Namespace: "/",
	})
	if err := so.Emit(engine.EventMessage, MessageTypeString, b[0]); err != nil {
		return nil, err
	}
	return &Socket{
		so:      so,
		encoder: encoder,
		decoder: decoder,
		nsp:     make(map[string]*Namespace)}, nil
}

func (s *Socket) namespace(nsp string) *Namespace {
	if nsp == "" {
		nsp = "/"
	}
	s.mutex.RLock()
	n, ok := s.nsp[nsp]
	s.mutex.RUnlock()
	if !ok {
		n = newNamespace(nsp)
		s.mutex.Lock()
		s.nsp[nsp] = n
		s.mutex.Unlock()
	}
	return n
}

func (s *Socket) Emit(nsp string, event string, args ...interface{}) (err error) {
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

func (s *Socket) ack(p *Packet) (err error) {
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

func (s *Socket) On(nsp string, event string, callback interface{}) {
	s.namespace(nsp).On(event, callback)
}

func (s *Socket) fire(nsp string, event string, args []byte, buffer [][]byte) ([]reflect.Value, error) {
	return s.namespace(nsp).fire(event, args, buffer)
}

func (s *Socket) process(p *Packet) {
	switch p.Type {
	case PacketTypeConnect:
		s.fire(p.Namespace, "connect", nil, nil) // client
	case PacketTypeDisconnect:
		s.Close()
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

func (s *Socket) yield() *Packet {
	select {
	case p := <-s.decoder.Decoded():
		return p
	default:
		return nil
	}
}

func (s *Socket) Close() (err error) {
	return s.so.Close()
}

func (s *Socket) OnError(fn func(err error)) {
	s.onError = fn
}
