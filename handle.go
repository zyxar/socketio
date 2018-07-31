package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"
)

type namespace struct {
	callbacks    map[string]*callback
	onError      func(so Socket, err ...interface{})
	onDisconnect func(so Socket)
}

type Namespace interface {
	OnEvent(event string, callback interface{}) Namespace     // chainable
	OnDisconnect(fn func(so Socket)) Namespace                // chainable
	OnError(fn func(so Socket, err ...interface{})) Namespace // chainable
}

func (e *namespace) OnDisconnect(fn func(so Socket)) Namespace { e.onDisconnect = fn; return e }

func (e *namespace) OnError(fn func(so Socket, err ...interface{})) Namespace {
	e.onError = fn
	return e
}

func (e *namespace) OnEvent(event string, callback interface{}) Namespace {
	e.callbacks[event] = newCallback(callback)
	return e
}

func (e *namespace) fireEvent(so Socket, event string, args []byte, buffer [][]byte, au ArgsUnmarshaler) ([]reflect.Value, error) {
	fn, ok := e.callbacks[event]
	if ok {
		return fn.Call(so, au, args, buffer)
	}
	return nil, nil
}

type ackHandle struct {
	id     uint64
	ackmap map[uint64]*callback
	mutex  sync.RWMutex
}

func (a *ackHandle) fireAck(so Socket, id uint64, data []byte, buffer [][]byte, au ArgsUnmarshaler) (err error) {
	a.mutex.RLock()
	fn, ok := a.ackmap[id]
	a.mutex.RUnlock()
	if ok {
		a.mutex.Lock()
		delete(a.ackmap, id)
		a.mutex.Unlock()
		_, err = fn.Call(so, au, data, buffer)
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
