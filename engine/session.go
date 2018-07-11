package engine

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type session struct {
	*Socket
	id      string
	barrier atomic.Value
}

func (s *session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := s.Socket.Conn.(http.Handler); ok {
		handler.ServeHTTP(w, r)
	}
}

func newSession(conn Conn, emitter *emitter, readTimeout, writeTimeout time.Duration) *session {
	id := generateRandomKey(24)
	s := &session{
		Socket: &Socket{
			Conn:          conn,
			eventHandlers: newEventHandlers(),
			emitter:       emitter,
			readTimeout:   readTimeout,
			writeTimeout:  writeTimeout},
		id: base64.StdEncoding.EncodeToString(id),
	}
	pauseChan := make(chan struct{})
	close(pauseChan)
	s.barrier.Store(pauseChan)
	return s
}

func (s *session) Pause() {
	pauseChan := make(chan struct{})
	s.barrier.Store(pauseChan)
}

func (s *session) Resume() {
	close(s.barrier.Load().(chan struct{}))
}

func (s *session) CheckPaused() {
	select {
	case <-s.barrier.Load().(chan struct{}):
	}
}

type sessionManager struct {
	ß map[string]*session
	sync.RWMutex
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		ß: make(map[string]*session),
	}
}

func (s *sessionManager) Get(id string) (ß *session, b bool) {
	s.RLock()
	ß, b = s.ß[id]
	s.RUnlock()
	return
}

func (s *sessionManager) Remove(id string) {
	s.Lock()
	delete(s.ß, id)
	s.Unlock()
}

func (s *sessionManager) NewSession(conn Conn, emitter *emitter, readTimeout, writeTimeout time.Duration) *session {
	ß := newSession(conn, emitter, readTimeout, writeTimeout)
	s.Lock()
	s.ß[ß.id] = ß
	s.Unlock()
	return ß
}

func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}
