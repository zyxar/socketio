package engio

import (
	"encoding/json"
)

type Socket struct {
	Conn
	*eventHandlers
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
	err = send(s.Conn, MessageTypeString, pktType, data)
	return
}

func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, args)
}

func send(conn Conn, msgType MessageType, pktType PacketType, data []byte) (err error) {
	wc, err := conn.NextWriter(msgType, pktType)
	if err != nil {
		return
	}
	if len(data) > 0 {
		if _, err = wc.Write(data); err != nil {
			wc.Close()
			return
		}
	}
	return wc.Close()
}
