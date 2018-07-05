package engio

import (
	"encoding/json"
	"time"
)

type Socket struct {
	Conn
	*eventHandlers
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (s *Socket) Handle() error {
	return s.eventHandlers.handle(s)
}

func (s *Socket) Emit(event string, args interface{}) (err error) {
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
				s.fire(s, EventPing, MessageTypeString, nil)
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
	err = s.Conn.WritePacket(&Packet{MessageTypeString, pktType, data})
	return
}

func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, args)
}
