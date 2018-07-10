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
	sync.RWMutex
}

func (s *Socket) Upgrade(acceptor Acceptor, newConn Conn) {
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
	s.transport = acceptor.Transport()
	s.Unlock()
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
		defer func() {
			if err == nil {
				s.fire(s, EventPing, msgType, nil)
			}
		}()
	case EventPong:
		pktType = PacketTypePong
	default:
		err = ErrInvalidEvent
		return
	}
	data, err := json.Marshal(args)
	if err != nil {
		return
	}
	s.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	err = s.Conn.WritePacket(&Packet{msgType, pktType, data})
	return
}

func (s *Socket) Emit(event event, args interface{}) (err error) {
	return s.emit(event, MessageTypeString, args)
}

func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, args)
}
