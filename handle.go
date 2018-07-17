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
		ackHandle: ackHandle{},
	}
}

type eventHandle struct {
	handlers map[string]*handleFn
	mutex    sync.RWMutex
}

func (e *eventHandle) On(event string, callback interface{}) {
	e.mutex.Lock()
	e.handlers[event] = newHandleFn(callback)
	e.mutex.Unlock()
}

func (e *eventHandle) fire(event string, args []byte, buffer [][]byte) ([]reflect.Value, error) {
	e.mutex.RLock()
	fn, ok := e.handlers[event]
	e.mutex.RUnlock()
	if ok {
		return fn.Call(args, buffer)
	}
	return nil, nil
}

type ackHandle struct {
	id     uint64
	ackmap sync.Map
}

func (a *ackHandle) onAck(id uint64, data []byte, buffer [][]byte) {
	if fn, ok := a.ackmap.Load(id); ok {
		a.ackmap.Delete(id)
		fn.(*handleFn).Call(data, buffer)
	}
}

func (a *ackHandle) store(callback interface{}) uint64 {
	id := atomic.AddUint64(&a.id, 1)
	a.ackmap.Store(id, newHandleFn(callback))
	return id
}
