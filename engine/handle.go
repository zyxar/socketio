package engine

import (
	"sync"
	"time"
)

type event string

const (
	EventOpen    event = "open"    // Fired upon successful connection.
	EventMessage event = "message" // Fired when data is received from the server.
	EventClose   event = "close"   // Fired upon disconnection. In compliance with the WebSocket API spec, this event may be fired even if the open event does not occur (i.e. due to connection error or close()).
	EventError   event = "error"   // Fired when an error occurs.
	EventUpgrade event = "upgrade" // Fired upon upgrade success, after the new transport is set
	EventPing    event = "ping"    // Fired upon flushing a ping packet (ie: actual packet write out)
	EventPong    event = "pong"    // Fired upon receiving a pong packet.
)

// Callback is a Callable func, default event handler
type Callback func(typ MessageType, data []byte)

// Callable is event handle to be called when event occurs
type Callable interface {
	Call(typ MessageType, data []byte)
}

// Call implements Callable interface
func (h Callback) Call(typ MessageType, data []byte) {
	h(typ, data)
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

func (e *eventHandlers) fire(event event, typ MessageType, data []byte) {
	e.RLock()
	callable, ok := e.handlers[event]
	e.RUnlock()
	if ok {
		callable.Call(typ, data)
	}
}

func (e *eventHandlers) handle(so *Socket) error {
	so.RLock()
	so.SetReadDeadline(time.Now().Add(so.readTimeout))
	p, err := so.ReadPacket()
	so.RUnlock()
	if err != nil {
		return err
	}
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		e.fire(EventClose, p.msgType, p.data)
		return so.Close()
	case PacketTypePing:
		so.Emit(EventPong, p.msgType, p.data)
		e.fire(EventPing, p.msgType, p.data)
	case PacketTypePong:
		e.fire(EventPong, p.msgType, p.data)
	case PacketTypeMessage:
		e.fire(EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
	case PacketTypeNoop:
		// noop
	default:
		return ErrInvalidPayload
	}
	return nil
}
