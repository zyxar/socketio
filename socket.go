package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/zyxar/socketio/engine"
)

type Socket struct {
	so      *engine.Socket
	encoder Encoder
	decoder Decoder

	id     uint64
	ackmap sync.Map

	handlers map[string]*handleFn
	onError  func(err error)
	sync.RWMutex
}

func newSocket(so *engine.Socket, parser Parser) (*Socket, error) {
	encoder := parser.Encoder()
	decoder := parser.Decoder()
	b, _ := encoder.Encode(&Packet{
		Type:      PacketTypeConnect,
		Namespace: "/",
	})
	if err := so.Emit(engine.EventMessage, b); err != nil {
		return nil, err
	}
	return &Socket{
		so:       so,
		encoder:  encoder,
		decoder:  decoder,
		handlers: make(map[string]*handleFn)}, nil
}

func (s *Socket) Emit(event string, args ...interface{}) (err error) {
	data := []interface{}{event}
	pkt := &Packet{
		Type:      PacketTypeEvent,
		Namespace: "/",
	}
	for i := range args {
		if t := reflect.TypeOf(args[i]); t.Kind() == reflect.Func {
			id := s.genid()
			s.ackmap.Store(id, newHandleFn(args[i]))
			pkt.ID = newid(id)
		} else {
			data = append(data, args[i])
		}
	}
	pkt.Data = data
	b, _ := s.encoder.Encode(pkt)
	return s.so.Emit(engine.EventMessage, b)
}

func (s *Socket) Ack(pkt *Packet) (err error) {
	pkt.Type = PacketTypeAck
	b, err := s.encoder.Encode(pkt)
	if err != nil {
		return
	}
	return s.so.Emit(engine.EventMessage, b)
}

func (s *Socket) onAck(id uint64, data []byte) {
	if fn, ok := s.ackmap.Load(id); ok {
		s.ackmap.Delete(id)
		fn.(*handleFn).Call(data)
	}
}

func (s *Socket) On(event string, callback interface{}) {
	s.Lock()
	s.handlers[event] = newHandleFn(callback)
	s.Unlock()
}

func (s *Socket) fire(event string, args []byte) ([]reflect.Value, error) {
	s.RLock()
	fn, ok := s.handlers[event]
	s.RUnlock()
	if ok {
		return fn.Call(args)
	}
	return nil, nil
}

func (s *Socket) process(p *Packet) {
	switch p.Type {
	case PacketTypeConnect:
	case PacketTypeDisconnect:
		s.Close()
	case PacketTypeEvent, PacketTypeBinaryEvent:
		if p.event != nil {
			v, err := s.fire(p.event.name, p.event.data)
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
				if err = s.Ack(p); err != nil {
					if s.onError != nil {
						s.onError(err)
					}
				}
			}
		}
	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil && p.event != nil {
			s.onAck(*p.ID, p.event.data)
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

func (s *Socket) genid() uint64 {
	return atomic.AddUint64(&s.id, 1)
}

func (s *Socket) OnError(fn func(err error)) {
	s.onError = fn
}
