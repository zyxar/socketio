package engine

import (
	"sync"
)

type event string

const (
	// EventOpen is fired upon successful connection.
	EventOpen event = "open"
	// EventMessage is fired when data is received from the server.
	EventMessage event = "message"
	// EventClose is fired upon disconnection. In compliance with the WebSocket API spec, this event may be fired even if the open event does not occur (i.e. due to connection error or close()).
	EventClose event = "close"
	// EventError is fired when an error occurs.
	EventError event = "error"
	// EventUpgrade is fired upon upgrade success, after the new transport is set
	EventUpgrade event = "upgrade"
	// EventPing is fired upon flushing a ping packet (ie: actual packet write out)
	EventPing event = "ping"
	// EventPong is fired upon receiving a pong packet.
	EventPong event = "pong"
)

// Callback is a Callable func, default event handler
type Callback func(so *Socket, typ MessageType, data []byte)

// Callable is event handle to be called when event occurs
type Callable interface {
	Call(so *Socket, typ MessageType, data []byte)
}

// Call implements Callable interface
func (h Callback) Call(so *Socket, typ MessageType, data []byte) {
	h(so, typ, data)
}

type eventHandlers struct {
	handlers map[event]Callable
	sync.RWMutex
}

func newEventHandlers() *eventHandlers {
	return &eventHandlers{
		handlers: make(map[event]Callable),
	}
}

func (e *eventHandlers) On(event event, callable Callable) {
	e.Lock()
	e.handlers[event] = callable
	e.Unlock()
}

func (e *eventHandlers) fire(so *Socket, event event, typ MessageType, data []byte) {
	e.RLock()
	callable, ok := e.handlers[event]
	e.RUnlock()
	if ok {
		callable.Call(so, typ, data)
	}
}
