package engine

import (
	"errors"
	"sync"
	"time"
)

var (
	errEmitterClosed = errors.New("emitter closed")
)

type emitter struct {
	so      *Socket
	packets chan *Packet
	done    chan struct{}
	wg      sync.WaitGroup
}

func newEmitter(so *Socket, bufSize int) *emitter {
	e := &emitter{
		so:      so,
		packets: make(chan *Packet, bufSize),
		done:    make(chan struct{}),
	}
	e.wg.Add(1)
	go e.loop()
	return e
}

func (e *emitter) close() {
	close(e.done)
	e.wg.Wait()
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
	defer e.wg.Done()
	defer e.flush()
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
			e.so.barrier.Wait()
			e.so.RLock()
			e.so.SetWriteDeadline(time.Now().Add(e.so.writeTimeout))
			e.so.WritePacket(p)
			e.so.RUnlock()
		}
	}
}

func (e *emitter) flush() {
	for {
		select {
		case p := <-e.packets:
			e.so.barrier.Wait()
			e.so.RLock()
			e.so.SetWriteDeadline(time.Now().Add(e.so.writeTimeout))
			e.so.WritePacket(p)
			e.so.RUnlock()
		default:
			return
		}
	}
}
