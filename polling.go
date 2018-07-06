package engio

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPollingConnClosed       = errors.New("polling connection closed")
	ErrPollingConnReadTimeout  = errors.New("polling connection read timeout")
	ErrPollingConnWriteTimeout = errors.New("polling connection write timeout")
	ErrPollingConnPaused       = errors.New("polling connection paused")
)

type pollingConn struct {
	in            chan *Packet
	out           chan *Packet
	closed        chan struct{}
	once          sync.Once
	readDeadline  atomic.Value
	writeDeadline atomic.Value
	paused        atomic.Value
}

func (p *pollingConn) Close() error {
	p.once.Do(func() {
		close(p.closed)
	})
	return nil
}

func NewPollingConn(bufSize int) *pollingConn {
	p := &pollingConn{
		in:     make(chan *Packet, bufSize),
		out:    make(chan *Packet, bufSize),
		closed: make(chan struct{}),
	}
	p.paused.Store(make(chan struct{}))
	return p
}

func (p *pollingConn) ReadPacket() (*Packet, error) {
	if p.isClosed() {
		return nil, ErrPollingConnClosed
	}
	var timer <-chan time.Time
	t := p.readDeadline.Load()
	if t != nil {
		deadline := t.(time.Time)
		timeout := deadline.Sub(time.Now())
		if timeout > 0 {
			timer = time.After(timeout)
		}
	}
	select {
	case <-p.closed:
		return nil, ErrPollingConnClosed
	case pkt, ok := <-p.in:
		if !ok {
			return nil, ErrPollingConnClosed
		}
		return pkt, nil
	case <-p.pauseChan():
		return nil, ErrPollingConnPaused
	case <-timer:
		return nil, ErrPollingConnReadTimeout
	}
}

func (p *pollingConn) ReadPacketOut() (*Packet, error) {
	if p.isClosed() {
		return nil, ErrPollingConnClosed
	}
	select {
	case <-p.closed:
		return nil, ErrPollingConnClosed
	case pkt, ok := <-p.out:
		if !ok {
			return nil, ErrPollingConnClosed
		}
		return pkt, nil
	case <-p.pauseChan():
		return &Packet{msgType: MessageTypeString, pktType: PacketTypeNoop}, nil
	}
}

func (p *pollingConn) WritePacket(pkt *Packet) error {
	if p.isClosed() {
		return ErrPollingConnClosed
	}
	var timer <-chan time.Time
	t := p.writeDeadline.Load()
	if t != nil {
		deadline := t.(time.Time)
		timeout := deadline.Sub(time.Now())
		if timeout > 0 {
			timer = time.After(timeout)
		}
	}
	select {
	case <-p.closed:
		return ErrPollingConnClosed
	case p.out <- pkt:
	case <-timer:
		return ErrPollingConnWriteTimeout
	case <-p.pauseChan():
		return ErrPollingConnPaused
	}
	return nil
}

func (p *pollingConn) SetReadDeadline(t time.Time) error {
	p.readDeadline.Store(t)
	return nil
}

func (p *pollingConn) SetWriteDeadline(t time.Time) error {
	p.writeDeadline.Store(t)
	return nil
}

func (p *pollingConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		pkt, err := p.ReadPacketOut()
		if err != nil {
			http.Error(w, err.Error(), http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusOK)
		q := r.URL.Query()
		if jsonp := q.Get(queryJSONP); jsonp != "" {
			err = writeJSONP(w, jsonp, pkt)
		} else {
			err = writeXHR(w, pkt)
		}
		if err != nil {
			println(err.Error())
		}
	case "POST":
		var payload Payload
		_, err := payload.ReadFrom(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for i := range payload.packets {
			select {
			case <-p.closed:
				http.Error(w, "closed", http.StatusGone)
				return
			case <-p.pauseChan():
				http.Error(w, "paused", http.StatusBadGateway)
				return
			case p.in <- &payload.packets[i]:
			}
		}
		http.Error(w, "OK", http.StatusOK)
	default:
		http.Error(w, "error", http.StatusMethodNotAllowed)
	}
}

func (p *pollingConn) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
	}
	return false
}

func (p *pollingConn) pauseChan() <-chan struct{} {
	return p.paused.Load().(chan struct{})
}

func (p *pollingConn) Pause() error {
	close(p.paused.Load().(chan struct{}))
	return nil
}
func (p *pollingConn) Resume() error {
	p.paused.Store(make(chan struct{}))
	return nil
}

var _ Conn = NewPollingConn(1)

func writeJSONP(w http.ResponseWriter, jsonp string, wt io.WriterTo) error {
	var buf bytes.Buffer
	w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
	if _, err := wt.WriteTo(&buf); err != nil {
		return err
	}
	tmpl := template.JSEscapeString(buf.String())
	_, err := fmt.Fprintf(w, `___eio[%s]("%s");`, jsonp, tmpl)
	return err
}

func writeXHR(w http.ResponseWriter, wt io.WriterTo) error {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := wt.WriteTo(w); err != nil {
		return err
	}
	return nil
}
