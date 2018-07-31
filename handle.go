package socketio

import (
	"reflect"
	"sync"
	"sync/atomic"
)

type namespace struct {
	callbacks    map[string]*callback
	onConnect    func(so Socket)
	onDisconnect func(so Socket)
	onError      func(so Socket, err ...interface{})
}

// Namespace is socket.io `namespace` abstraction
type Namespace interface {
	// OnEvent registers event callback:
	// callback should be a valid function, 1st argument of which could be `socketio.Socket` or omitted;
	// the event callback would be called when a message received from a client with corresponding event;
	// upon invocation the corresponding `socketio.Socket` would be supplied if appropriate.
	OnEvent(event string, callback interface{}) Namespace // chainable
	// OnConnect registers fn as callback, which would be called when this Namespace is connected by a
	// client, i.e. upon receiving CONNECT packet (for non-root namespace) or connection establishment
	// ("/" namespace)
	OnConnect(fn func(so Socket)) Namespace // chainable
	// OnDisconnect registers fn as callback, which would be called when this Namespace is disconnected by a
	// client, i.e. upon receiving DISCONNECT packet or connection lost
	OnDisconnect(fn func(so Socket)) Namespace // chainable
	// OnError registers fn as callback, which would be called when error occurs in this Namespace
	OnError(fn func(so Socket, err ...interface{})) Namespace // chainable
}

func (e *namespace) OnDisconnect(fn func(so Socket)) Namespace { e.onDisconnect = fn; return e }
func (e *namespace) OnConnect(fn func(so Socket)) Namespace    { e.onConnect = fn; return e }

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
