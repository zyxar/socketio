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
)

type pollingConn struct {
	in            chan *Packet
	out           chan *Packet
	closed        chan struct{}
	once          sync.Once
	wg            sync.WaitGroup
	readDeadline  atomic.Value
	writeDeadline atomic.Value
}

func (p *pollingConn) close() {
	p.once.Do(func() {
		close(p.closed)
		close(p.in)
		close(p.out)
	})
}

func (p *pollingConn) Close() error {
	p.close()
	p.wg.Wait()
	return nil
}

func NewPollingConn(bufSize int) *pollingConn {
	p := &pollingConn{
		in:     make(chan *Packet, bufSize),
		out:    make(chan *Packet, bufSize),
		closed: make(chan struct{}),
	}
	return p
}

func (p *pollingConn) ReadPacket() (*Packet, error) {
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
	case <-timer:
		return nil, ErrPollingConnReadTimeout
	}
}

func (p *pollingConn) ReadPacketOut() (*Packet, error) {
	select {
	case <-p.closed:
		return nil, ErrPollingConnClosed
	case pkt, ok := <-p.out:
		if !ok {
			return nil, ErrPollingConnClosed
		}
		return pkt, nil
	}
}

func (p *pollingConn) WritePacket(pkt *Packet) error {
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
			case p.in <- &payload.packets[i]:
			}
		}
		http.Error(w, "OK", http.StatusOK)
	default:
		http.Error(w, "error", http.StatusMethodNotAllowed)
	}
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