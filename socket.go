package socketio

import (
	"sync"

	"github.com/zyxar/socketio/engine"
)

type Socket struct {
	so     *engine.Socket
	parser Parser

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
	data = append(data, args...)
	b, _ := s.parser.Encode(&Packet{
		Type:      PacketTypeEvent,
		Namespace: "/",
		Data:      data,
	})
	return s.so.Emit(engine.EventMessage, b)
}

func (s *Socket) On(event string, callback interface{}) {
	s.Lock()
	s.handlers[event] = newHandleFn(callback)
	s.Unlock()
}

func (s *Socket) fire(event string, args []byte) {
	s.RLock()
	fn, ok := s.handlers[event]
	s.RUnlock()
	if ok {
		fn.Call(args)
	}
}

func (s *Socket) Close() error {
	return s.so.Close()
}
