package engine

import (
	"time"
)

type object struct {
	conn    Conn
	p       *Packet
	timeout time.Duration
}

type emitter struct {
	oc   chan *object
	done <-chan struct{}
}

func newEmitter(bufSize int, done <-chan struct{}) *emitter {
	return &emitter{
		oc:   make(chan *object, bufSize),
		done: done,
	}
}

func (e *emitter) submit(o *object) {
	select {
	case <-e.done:
	case e.oc <- o:
	}
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
		case o := <-e.oc:
			o.conn.SetWriteDeadline(time.Now().Add(o.timeout))
			o.conn.WritePacket(o.p)
		}
	}
}
