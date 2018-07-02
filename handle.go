package engio

import (
	"bytes"
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

type Handle func(so *Socket, data []byte)

type Callable interface {
	Call(so *Socket, data []byte)
}

func (h Handle) Call(so *Socket, data []byte) {
	h(so, data)
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

func (e *eventHandlers) fire(so *Socket, event string, data []byte) {
	e.RLock()
	callable, ok := e.handlers[event]
	e.RUnlock()
	if ok {
		callable.Call(so, data)
	}
}

func (e *eventHandlers) handle(so *Socket) error {
	_, packetType, rc, err := so.NextReader()
	if err != nil {
		return err
	}
	defer rc.Close()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(rc); err != nil {
		return err
	}
	switch packetType {
	case PacketTypeOpen:
	case PacketTypeClose:
	case PacketTypePing:
	case PacketTypePong:
	case PacketTypeMessage:
		e.fire(so, EventMessage, buffer.Bytes())
	case PacketTypeUpgrade:
		e.fire(so, EventUpgrade, buffer.Bytes())
	case PacketTypeNoop:
		// noop
	default:
		return ErrInvalidMessage
	}
	return nil
}
