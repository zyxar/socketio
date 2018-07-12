package engine

import (
	"errors"
	"time"
)

var (
	errEmitterClosed = errors.New("emitter closed")
)

type emitter struct {
	so      *Socket
	packets chan *Packet
	done    chan struct{}
}

func newEmitter(so *Socket, bufSize int) *emitter {
	return &emitter{
		so:      so,
		packets: make(chan *Packet, bufSize),
		done:    make(chan struct{}),
	}
}

func (e *emitter) close() {
	close(e.done)
}
func (e *emitter) submit(p *Packet) error {
	select {
	case <-e.done:
		return errEmitterClosed
	default:
	}
	select {
	case <-e.done:
		return errEmitterClosed
	case e.packets <- p:
	}
	return nil
}

func (e *emitter) loop() {
	for {
		select {
		case <-e.done:
			return
		default:
		}
		select {
		case <-e.done:
			return
		case p := <-e.packets:
			e.so.CheckPaused()
			e.so.RLock()
			e.so.SetWriteDeadline(time.Now().Add(e.so.writeTimeout))
			e.so.WritePacket(p)
			e.so.RUnlock()
		}
	}
}
