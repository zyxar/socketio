package engine

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// Server is engine.io server implementation
type Server struct {
	pingInterval time.Duration
	pingTimeout  time.Duration
	ßchan        chan *Socket
	done         chan struct{}
	once         sync.Once
	*sessionManager
	*eventHandlers
}

// NewServer creates a enine.io server instance
func NewServer(interval, timeout time.Duration, onOpen func(*Socket)) (*Server, error) {
	done := make(chan struct{})
	s := &Server{
		pingInterval:   interval,
		pingTimeout:    timeout,
		ßchan:          make(chan *Socket, 1),
		done:           done,
		sessionManager: newSessionManager(),
		eventHandlers:  newEventHandlers(),
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case ß, ok := <-s.ßchan:
				if !ok {
					return
				}
				ß.Emit(EventOpen, MessageTypeString, &Parameters{
					SID:          ß.id,
					Upgrades:     []string{"websocket"},
					PingInterval: int(interval / time.Millisecond),
					PingTimeout:  int(timeout / time.Millisecond),
				})
				go func() {
					defer ß.Close()
					defer s.sessionManager.Remove(ß.id)
					var p *Packet
					var err error
					for {
						if p, err = ß.Read(); err != nil {
							if err == ErrPollingConnPaused {
								ß.barrier.Wait()
								continue
							}
							log.Println("handle:", err.Error())
							s.fire(ß, EventClose, MessageTypeString, nil)
							return
						}
						if err = s.handle(ß, p); err != nil {
							log.Println("handle:", err.Error())
						}
					}
				}()

				onOpen(ß)
			}
		}
	}()
	return s, nil
}

func (s *Server) handle(ß *Socket, p *Packet) (err error) {
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		s.fire(ß, EventClose, p.msgType, p.data)
		return ß.Close()
	case PacketTypePing:
		err = ß.Emit(EventPong, p.msgType, p.data)
		s.fire(ß, EventPing, p.msgType, p.data)
	case PacketTypePong:
		s.fire(ß, EventPong, p.msgType, p.data)
	case PacketTypeMessage:
		s.fire(ß, EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
	case PacketTypeNoop:
	default:
		return ErrInvalidPayload
	}
	return
}

// Close signals stop to background workers and closes server
func (s *Server) Close() (err error) {
	s.once.Do(func() {
		close(s.done)
	})
	return
}

// ServeHTTP impements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get(queryEIO) != Version {
		http.Error(w, "protocol version incompatible", http.StatusBadRequest)
		return
	}

	transport := getTransport(query.Get(queryTransport))
	if transport == nil {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}

	if sid := query.Get(querySession); sid == "" {
		conn, err := transport.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ß := s.NewSession(conn, s.pingTimeout+s.pingInterval, s.pingTimeout)
		ß.transportName = transport.Name()
		select {
		case <-s.done:
			return
		case s.ßchan <- ß:
		}
		ß.ServeHTTP(w, r)
	} else {
		ß, exists := s.sessionManager.Get(sid)
		if !exists {
			http.Error(w, "invalid session", http.StatusBadRequest)
			return
		}
		if transportName := transport.Name(); ß.transportName != transportName {
			conn, err := transport.Accept(w, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.upgrade(ß, transportName, conn)
		}
		ß.ServeHTTP(w, r)
	}
}

func (s *Server) upgrade(ß *Socket, transportName string, newConn Conn) {
	defer ß.barrier.Pause().Resume()

	newConn.SetReadDeadline(time.Now().Add(ß.readTimeout))
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
	newConn.SetWriteDeadline(time.Now().Add(ß.writeTimeout))
	if err = newConn.WritePacket(p); err != nil {
		newConn.Close()
		return
	}

	ß.RLock()
	conn := ß.Conn
	ß.RUnlock()

	if err := conn.Pause(); err != nil {
		newConn.Close()
		return
	}

	newConn.SetReadDeadline(time.Now().Add(ß.readTimeout))
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

	for _, packet := range conn.FlushOut() {
		newConn.SetWriteDeadline(time.Now().Add(ß.writeTimeout))
		newConn.WritePacket(packet)
	}

	ß.Lock()
	ß.Conn = newConn
	ß.transportName = transportName
	ß.Unlock()

	for _, packet := range conn.FlushIn() {
		s.handle(ß, packet)
	}

	s.fire(ß, EventUpgrade, p.msgType, p.data)
}
