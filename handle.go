package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"
)

type nspHandle struct {
	eventHandle
	ackHandle
}

func newNspHandle(namespace string) *nspHandle {
	return &nspHandle{
		eventHandle: eventHandle{
			handlers: make(map[string]*handleFn),
		},
		ackHandle: ackHandle{ackmap: make(map[uint64]*handleFn)},
	}
}

type eventHandle struct {
	handlers map[string]*handleFn
	mutex    sync.RWMutex
}

func (e *eventHandle) onEvent(event string, callback interface{}) {
	e.mutex.Lock()
	e.handlers[event] = newHandleFn(callback)
	e.mutex.Unlock()
}

func (e *eventHandle) fireEvent(event string, args []byte, buffer [][]byte, au ArgsUnmarshaler) ([]reflect.Value, error) {
	e.mutex.RLock()
	fn, ok := e.handlers[event]
	e.mutex.RUnlock()
	if ok {
		return fn.Call(au, args, buffer)
	}
	return nil, nil
}

type ackHandle struct {
	id     uint64
	ackmap map[uint64]*handleFn
	mutex  sync.RWMutex
}

func (a *ackHandle) fireAck(id uint64, data []byte, buffer [][]byte, au ArgsUnmarshaler) (err error) {
	a.mutex.RLock()
	fn, ok := a.ackmap[id]
	a.mutex.RUnlock()
	if ok {
		a.mutex.Lock()
		delete(a.ackmap, id)
		a.mutex.Unlock()
		_, err = fn.Call(au, data, buffer)
	}
	return
}

func (a *ackHandle) onAck(callback interface{}) uint64 {
	id := atomic.AddUint64(&a.id, 1)
	a.mutex.Lock()
	a.ackmap[id] = newHandleFn(callback)
	a.mutex.Unlock()
	return id
}
