package engine

import (
	"context"
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
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	*sessionManager
	*eventHandlers
}

// NewServer creates a enine.io server instance
func NewServer(ctx context.Context, interval, timeout time.Duration, onOpen func(*Socket)) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &Server{
		pingInterval:   interval,
		pingTimeout:    timeout,
		ßchan:          make(chan *Socket, 1),
		ctx:            ctx,
		cancel:         cancel,
		sessionManager: newSessionManager(),
		eventHandlers:  newEventHandlers(),
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
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
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					defer ß.Close()
					defer s.sessionManager.Remove(ß.id)
					var p *Packet
					var err error
					for {
						if p, err = ß.Read(); err != nil {
							if err == ErrPollingConnPaused {
								ß.CheckPaused()
								continue
							}
							select {
							case <-ctx.Done():
								return
							default:
							}
							log.Println("engine.io read:", err.Error())
							s.fire(ß, EventClose, MessageTypeString, nil)
							return
						}
						if err = s.handle(ß, p); err != nil {
							select {
							case <-ctx.Done():
								return
							default:
							}
							log.Println("engine.io handle:", err.Error())
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
	s.cancel()
	s.wg.Wait()
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
		ß := s.NewSession(s.ctx, conn, s.pingTimeout+s.pingInterval, s.pingTimeout)
		ß.transportName = transport.Name()
		select {
		case <-s.ctx.Done():
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
	ß.pause()
	defer ß.resume()
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
