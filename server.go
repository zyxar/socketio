package engio

import (
	"net/http"
	"sync"
	"time"
)

type Server struct {
	pingInterval time.Duration
	pingTimeout  time.Duration
	ßchan        chan *session
	once         sync.Once
	*sessionManager
	*eventHandlers
}

func NewServer(interval, timeout time.Duration) (*Server, error) {
	s := &Server{
		pingInterval:   interval,
		pingTimeout:    timeout,
		ßchan:          make(chan *session, 1),
		sessionManager: newSessionManager(),
		eventHandlers:  newEventHandlers(),
	}

	go func() {
		for {
			select {
			case ß, ok := <-s.ßchan:
				if !ok {
					return
				}
				s.fire(ß.Socket, EventOpen, MessageTypeString, nil)
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
							println(err.Error())
							so.fire(so, EventClose, MessageTypeString, nil)
							return
						}
					}
				}()
			}
		}
	}()
	return s, nil
}

func (s *Server) Close() (err error) {
	s.once.Do(func() {
		close(s.ßchan)
	})
	return
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	println(r.Method, r.URL.RawQuery)
	acceptor := getAcceptor(query.Get(queryTransport))

	if acceptor == nil {
		http.Error(w, "invalid transport", http.StatusBadRequest)
		return
	}

	var ß *session
	sid := query.Get(querySession)
	if sid == "" {
		conn, err := acceptor.Accept(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ß = s.NewSession(conn, s.pingTimeout+s.pingInterval, s.pingTimeout)
		ß.transport = acceptor.Transport()
		ß.Emit(EventOpen, &Parameters{
			SID:          ß.id,
			Upgrades:     []string{"websocket"},
			PingInterval: int(s.pingInterval / time.Millisecond),
			PingTimeout:  int(s.pingTimeout / time.Millisecond),
		})
		s.ßchan <- ß
	} else {
		var exists bool
		ß, exists = s.sessionManager.Get(sid)
		if !exists {
			http.Error(w, "invalid session", http.StatusBadRequest)
			return
		}
		if ß.transport != acceptor.Transport() {
			conn, err := acceptor.Accept(w, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ß.Pause()
			ß.Upgrade(acceptor, conn)
			ß.Resume()
		}
	}
	ß.ServeHTTP(w, r)
	return
}

func (s *Server) BindAndListen(srv *http.Server) error {
	if srv == nil {
		panic("nil http server")
	}
	srv.Handler = s
	return srv.ListenAndServe()
}
