package engine

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Socket is engine.io connection encapsulation
type Socket struct {
	Conn
	*eventHandlers
	readTimeout   time.Duration
	writeTimeout  time.Duration
	transportName string
	id            string
	barrier       atomic.Value
	emitter       *emitter
	once          sync.Once
	sync.RWMutex
}

func newSocket(conn Conn, readTimeout, writeTimeout time.Duration, id string) *Socket {
	so := &Socket{
		Conn:          conn,
		eventHandlers: newEventHandlers(),
		readTimeout:   readTimeout,
		writeTimeout:  writeTimeout,
		id:            id}
	emitter := newEmitter(so, 8)
	go emitter.loop()
	so.emitter = emitter
	pauseChan := make(chan struct{})
	close(pauseChan)
	so.barrier.Store(pauseChan)
	return so
}

// CheckPaused blocks when socket is paused
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

	if packets := conn.FlushIn(); packets != nil {
		for _, packet := range packets {
			s.handle(packet)
		}
	}

	s.fire(EventUpgrade, p.msgType, p.data)
}

// Handle is event handling helper
func (s *Socket) Handle() (err error) {
	s.RLock()
	conn := s.Conn
	s.RUnlock()
	if err = conn.SetReadDeadline(time.Now().Add(s.readTimeout)); err != nil {
		return
	}
	p, err := conn.ReadPacket()
	if err != nil {
		return err
	}
	return s.handle(p)
}

func (s *Socket) handle(p *Packet) (err error) {
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		s.fire(EventClose, p.msgType, p.data)
		return s.Close()
	case PacketTypePing:
		err = s.Emit(EventPong, p.msgType, p.data)
		s.fire(EventPing, p.msgType, p.data)
	case PacketTypePong:
		s.fire(EventPong, p.msgType, p.data)
	case PacketTypeMessage:
		s.fire(EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
	case PacketTypeNoop:
		// noop
	default:
		return ErrInvalidPayload
	}
	return
}

// Close closes underlying connection and background emitter
func (s *Socket) Close() (err error) {
	s.once.Do(func() {
		s.emitter.close()
		err = s.Conn.Close()
	})
	return
}

// Emit sends event data to remote peer
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

// Send is short for Emitting message event
func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, MessageTypeString, args)
}

// ServeHTTP implements http.Handler
func (s *Socket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := s.Conn.(http.Handler); ok {
		handler.ServeHTTP(w, r)
	}
}

// Sid returns socket session id, assigned by server.
func (s *Socket) Sid() string {
	return s.id
}
