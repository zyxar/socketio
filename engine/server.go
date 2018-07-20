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
					for {
						if err := ß.Handle(); err != nil {
							if err == ErrPollingConnPaused {
								ß.CheckPaused()
								continue
							}
							log.Println("handle:", err.Error())
							ß.fire(EventClose, MessageTypeString, nil)
							return
						}
					}
				}()

				onOpen(ß)
			}
		}
	}()
	return s, nil
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
			ß.upgrade(transportName, conn)
		}
		ß.ServeHTTP(w, r)
	}
}
