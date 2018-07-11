package engine

import (
	"encoding/json"
	"sync"
	"time"
)

type Socket struct {
	Conn
	*eventHandlers
	readTimeout  time.Duration
	writeTimeout time.Duration
	transport    string
	*emitter
	sync.RWMutex
}

func (s *Socket) upgrade(transport string, newConn Conn) {
	newConn.SetReadDeadline(time.Now().Add(s.readTimeout))
	p, err := newConn.ReadPacket()
	if err != nil {
		newConn.Close()
		return
	}
	if p.pktType != PacketTypePing {
		newConn.Close()
		return
	}
	p.pktType = PacketTypePong
	newConn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	if err = newConn.WritePacket(p); err != nil {
		newConn.Close()
		return
	}

	s.RLock()
	conn := s.Conn
	s.RUnlock()

	if err := conn.Pause(); err != nil {
		newConn.Close()
		return
	}

	newConn.SetReadDeadline(time.Now().Add(s.readTimeout))
	p, err = newConn.ReadPacket()
	if err != nil {
		newConn.Close()
		conn.Resume()
		return
	}
	if p.pktType != PacketTypeUpgrade {
		newConn.Close()
		conn.Resume()
		return
	}

	s.Lock()
	s.Conn = newConn
	s.transport = transport
	s.Unlock()
	s.fire(s, EventUpgrade, p.msgType, p.data)
	return
}

func (s *Socket) Handle() error {
	return s.eventHandlers.handle(s)
}

func (s *Socket) emit(event event, msgType MessageType, args interface{}) (err error) {
	var pktType PacketType
	switch event {
	case EventOpen:
		pktType = PacketTypeOpen
	case EventMessage:
		pktType = PacketTypeMessage
	case EventClose:
		pktType = PacketTypeClose
	// case EventError:
	// case EventUpgrade:
	case EventPing:
		pktType = PacketTypePing
	case EventPong:
		pktType = PacketTypePong
	default:
		err = ErrInvalidEvent
		return
	}
	var data []byte
	if d, ok := args.([]byte); ok {
		data = d
	} else if s, ok := args.(string); ok {
		data = []byte(s)
	} else {
		data, err = json.Marshal(args)
		if err != nil {
			return
		}
	}

	s.RLock()
	object := &object{
		conn:    s.Conn,
		p:       &Packet{msgType, pktType, data},
		timeout: s.writeTimeout,
	}
	s.RUnlock()
	s.submit(object)
	return
}

func (s *Socket) Emit(event event, args interface{}) (err error) {
	return s.emit(event, MessageTypeString, args)
}

func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, args)
}
