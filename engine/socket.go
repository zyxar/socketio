package engine

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

type Socket struct {
	Conn
	*eventHandlers
	readTimeout   time.Duration
	writeTimeout  time.Duration
	transportName string
	barrier       atomic.Value
	emitter       *emitter
	once          sync.Once
	sync.RWMutex
}

func newSocket(conn Conn, readTimeout, writeTimeout time.Duration) *Socket {
	so := &Socket{
		Conn:          conn,
		eventHandlers: newEventHandlers(),
		readTimeout:   readTimeout,
		writeTimeout:  writeTimeout}
	emitter := newEmitter(so, 8)
	go emitter.loop()
	so.emitter = emitter
	so.pause()
	so.resume()
	return so
}

func (s *Socket) CheckPaused() {
	<-s.barrier.Load().(chan struct{})
}

func (s *Socket) pause() {
	pauseChan := make(chan struct{})
	s.barrier.Store(pauseChan)
}

func (s *Socket) resume() {
	close(s.barrier.Load().(chan struct{}))
}

func (s *Socket) upgrade(transportName string, newConn Conn) {
	s.pause()
	defer s.resume()
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

	conn.Close()
	if packets := conn.FlushOut(); packets != nil {
		newConn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		for _, packet := range packets {
			newConn.WritePacket(packet)
		}
	}

	s.Lock()
	s.Conn = newConn
	s.transportName = transportName
	s.Unlock()
	s.fire(EventUpgrade, p.msgType, p.data)
}

func (s *Socket) Handle() error {
	return s.eventHandlers.handle(s)
}

func (s *Socket) Close() (err error) {
	s.once.Do(func() {
		s.emitter.close()
		err = s.Conn.Close()
	})
	return
}

func (s *Socket) Emit(event event, msgType MessageType, args interface{}) (err error) {
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

	return s.emitter.submit(&Packet{msgType: msgType, pktType: pktType, data: data})
}

func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, MessageTypeString, args)
}
