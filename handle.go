package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"
)

type namespace struct{ callbacks map[string]*callback }

type Namespace interface {
	OnEvent(event string, callback interface{}) Namespace // chainable
}

func (e *namespace) OnEvent(event string, callback interface{}) Namespace {
	e.callbacks[event] = newCallback(callback)
	return e
}

func (e *namespace) fireEvent(event string, args []byte, buffer [][]byte, au ArgsUnmarshaler) ([]reflect.Value, error) {
	fn, ok := e.callbacks[event]
	if ok {
		return fn.Call(au, args, buffer)
	}
	return nil, nil
}

type ackHandle struct {
	id     uint64
	ackmap map[uint64]*callback
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
	a.ackmap[id] = newCallback(callback)
	a.mutex.Unlock()
	return id
}
