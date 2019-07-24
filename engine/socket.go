package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Socket is engine.io connection encapsulation
type Socket struct {
	Conn
	readTimeout   time.Duration
	writeTimeout  time.Duration
	transportName string
	id            string
	barrier       atomic.Value
	packets       chan *Packet
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closeOnce     sync.Once
	sync.RWMutex
}

func newSocket(ctx context.Context, conn Conn, readTimeout, writeTimeout time.Duration, id string) *Socket {
	ctx, cancel := context.WithCancel(ctx)
	so := &Socket{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		id:           id,
		packets:      make(chan *Packet, 8),
		ctx:          ctx,
		cancel:       cancel,
	}

	so.wg.Add(1)
	go so.loop()

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

// ContextErr returns non-nil error when Socket is closed, or context has been canceled; nil otherwise.
func (s *Socket) ContextErr() error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	return nil
}

// Close closes underlying connection and waits til corresponding routines exit
func (s *Socket) Close() (err error) {
	s.closeOnce.Do(func() {
		s.cancel()
		s.RLock()
		conn := s.Conn
		s.RUnlock()
		err = conn.Close()
		s.wg.Wait()
	})
	return
}

// Emit sends event data to remote peer
func (s *Socket) Emit(event event, msgType MessageType, args interface{}) (err error) {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}

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

	return s.submit(&Packet{msgType: msgType, pktType: pktType, data: data})
}

// Send is short for Emitting message event
func (s *Socket) Send(args interface{}) (err error) {
	return s.Emit(EventMessage, MessageTypeString, args)
}

// ServeHTTP implements http.Handler
func (s *Socket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	conn := s.Conn
	s.RUnlock()
	if handler, ok := conn.(http.Handler); ok {
		handler.ServeHTTP(w, r)
	}
}

// Sid returns socket session id, assigned by server.
func (s *Socket) Sid() string { return s.id }

func (s *Socket) submit(p *Packet) error {
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case s.packets <- p:
	}
	return nil
}

func (s *Socket) emit(p *Packet) {
	s.CheckPaused()
	s.RLock()
	conn := s.Conn
	s.RUnlock()
	conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
	conn.WritePacket(p)
}

func (s *Socket) loop() {
	defer s.wg.Done()
	defer s.flush()
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		select {
		case <-s.ctx.Done():
			return
		case p := <-s.packets:
			s.emit(p)
		}
	}
}

func (s *Socket) flush() {
	for {
		select {
		case p := <-s.packets:
			s.emit(p)
		default:
			return
		}
	}
}
