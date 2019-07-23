package engine

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"io"
	"sync"
	"time"
)

func newSession(ctx context.Context, conn Conn, readTimeout, writeTimeout time.Duration) *Socket {
	id := generateSidBytes(16)
	return newSocket(ctx, conn, readTimeout, writeTimeout, b32enc.EncodeToString(id))
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

func (s *sessionManager) NewSession(ctx context.Context, conn Conn, readTimeout, writeTimeout time.Duration) *Socket {
	ß := newSession(ctx, conn, readTimeout, writeTimeout)
	s.Lock()
	s.ß[ß.id] = ß
	s.Unlock()
	return ß
}

var b32enc = base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)

func generateSidBytes(length int) []byte { // length > 7
	now := uint64(time.Now().UnixNano() / int64(time.Millisecond))
	k := make([]byte, length)
	k[0] = byte(now >> 40)
	k[1] = byte(now >> 32)
	k[2] = byte(now >> 24)
	k[3] = byte(now >> 16)
	k[4] = byte(now >> 8)
	k[5] = byte(now)
	io.ReadFull(rand.Reader, k[6:])
	return k
}
