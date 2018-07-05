package engio

import (
	"sync"
)

const (
	EventOpen    = "open"    // Fired upon successful connection.
	EventMessage = "message" // Fired when data is received from the server.
	EventClose   = "close"   // Fired upon disconnection. In compliance with the WebSocket API spec, this event may be fired even if the open event does not occur (i.e. due to connection error or close()).
	EventError   = "error"   // Fired when an error occurs.
	EventUpgrade = "upgrade" // Fired upon upgrade success, after the new transport is set
	EventPing    = "ping"    // Fired upon flushing a ping packet (ie: actual packet write out)
	EventPong    = "pong"    // Fired upon receiving a pong packet.
)

type Callback func(so *Socket, typ MessageType, data []byte)

type Callable interface {
	Call(so *Socket, typ MessageType, data []byte)
}

func (h Callback) Call(so *Socket, typ MessageType, data []byte) {
	h(so, typ, data)
}

type eventHandlers struct {
	handlers map[string]Callable
	sync.RWMutex
}

func newEventHandlers() *eventHandlers {
	return &eventHandlers{
		handlers: make(map[string]Callable),
	}
}

func (e *eventHandlers) On(event string, callable Callable) {
	e.Lock()
	e.handlers[event] = callable
	e.Unlock()
}

func (e *eventHandlers) fire(so *Socket, event string, typ MessageType, data []byte) {
	e.RLock()
	callable, ok := e.handlers[event]
	e.RUnlock()
	if ok {
		callable.Call(so, typ, data)
	}
}

func (e *eventHandlers) handle(so *Socket) error {
	p, err := so.ReadPacket()
	if err != nil {
		return err
	}
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		e.fire(so, EventClose, p.msgType, p.data)
		return so.Close()
	case PacketTypePing:
		so.Emit(EventPong, p.data)
	case PacketTypePong:
		e.fire(so, EventPong, p.msgType, p.data)
	case PacketTypeMessage:
		e.fire(so, EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
		e.fire(so, EventUpgrade, p.msgType, p.data)
	case PacketTypeNoop:
		// noop
	default:
		return ErrInvalidMessage
	}
	return nil
}
