package engine

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Socket is engine.io connection encapsulation
type Socket struct {
	Conn
	readTimeout   time.Duration
	writeTimeout  time.Duration
	transportName string
	id            string
	barrier       Barrier
	emitter       *emitter
	once          sync.Once
	sync.RWMutex
}

func newSocket(conn Conn, readTimeout, writeTimeout time.Duration, id string) *Socket {
	so := &Socket{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		id:           id,
		barrier:      newLockBarrier()}
	so.emitter = newEmitter(so, 8)
	return so
}

// Read returns a Packet upon success or error on failure
func (s *Socket) Read() (p *Packet, err error) {
	s.RLock()
	conn := s.Conn
	s.RUnlock()
	if err = conn.SetReadDeadline(time.Now().Add(s.readTimeout)); err != nil {
		return
	}
	p, err = conn.ReadPacket()
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
