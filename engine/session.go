package engine

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"sync"
	"time"
)

func newSession(conn Conn, readTimeout, writeTimeout time.Duration) *Socket {
	id := generateRandomKey(24)
	return newSocket(conn, readTimeout, writeTimeout, base64.StdEncoding.EncodeToString(id))
}

type sessionManager struct {
	ß map[string]*Socket
	sync.RWMutex
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		ß: make(map[string]*Socket),
	}
}

func (s *sessionManager) Get(id string) (ß *Socket, b bool) {
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

func (s *sessionManager) NewSession(conn Conn, readTimeout, writeTimeout time.Duration) *Socket {
	ß := newSession(conn, readTimeout, writeTimeout)
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
