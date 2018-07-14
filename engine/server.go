package engine

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	pingInterval time.Duration
	pingTimeout  time.Duration
	ßchan        chan *session
	done         chan struct{}
	once         sync.Once
	*sessionManager
}

func NewServer(interval, timeout time.Duration, onOpen func(*Socket)) (*Server, error) {
	done := make(chan struct{})
	s := &Server{
		pingInterval:   interval,
		pingTimeout:    timeout,
		ßchan:          make(chan *session, 1),
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
					so := ß.Socket
					defer so.Close()
					defer s.sessionManager.Remove(ß.id)
					for {
						if err := so.Handle(); err != nil {
							if err == ErrPollingConnPaused {
								ß.CheckPaused()
								continue
							}
							log.Println("handle:", err.Error())
							so.fire(EventClose, MessageTypeString, nil)
							return
						}
					}
				}()

				onOpen(ß.Socket)
			}
		}
	}()
	return s, nil
}

func (s *Server) Close() (err error) {
	s.once.Do(func() {
		close(s.done)
	})
	return
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RemoteAddr, r.Method, r.URL.RawQuery)

	query := r.URL.Query()
	if query.Get(queryEIO) != Version {
		http.Error(w, "protocol version incompatible", http.StatusBadRequest)
		return
	}

	acceptor := getAcceptor(query.Get(queryTransport))
	if acceptor == nil {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}

	if sid := query.Get(querySession); sid == "" {
		conn, err := acceptor.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ß := s.NewSession(conn, s.pingTimeout+s.pingInterval, s.pingTimeout)
		ß.Socket.transport = acceptor.Transport()
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
		transport := acceptor.Transport()
		if ß.transport != transport {
			conn, err := acceptor.Accept(w, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ß.upgrade(transport, conn)
		}
		ß.ServeHTTP(w, r)
	}
	return
}
