package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/zyxar/socketio/engine"
)

type Socket struct {
	so     *engine.Socket
	parser Parser

	id     uint64
	ackmap sync.Map

	handlers map[string]*handleFn
	sync.RWMutex
}

func newSocket(so *engine.Socket, parser Parser) (*Socket, error) {
	b, _ := parser.Encode(&Packet{
		Type:      PacketTypeConnect,
		Namespace: "/",
	})
	if err := so.Emit(engine.EventMessage, b); err != nil {
		return nil, err
	}
	return &Socket{so: so, parser: parser, handlers: make(map[string]*handleFn)}, nil
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
	b, _ := s.parser.Encode(pkt)
	return s.so.Emit(engine.EventMessage, b)
}

func (s *Socket) Ack(pkt *Packet) (err error) {
	pkt.Type = PacketTypeAck
	b, err := s.parser.Encode(pkt)
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

func (s *Socket) Close() error {
	return s.so.Close()
}

func (s *Socket) genid() uint64 {
	return atomic.AddUint64(&s.id, 1)
}
