package engio

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
)

var (
	ErrPollingConnClosed = errors.New("polling connection closed")
)

type pollingConn struct {
	in     chan *Packet
	out    chan *Packet
	closed chan struct{}
	once   sync.Once
	wg     sync.WaitGroup
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
	select {
	case <-p.closed:
		return nil, ErrPollingConnClosed
	case pkt, ok := <-p.in:
		if !ok {
			return nil, ErrPollingConnClosed
		}
		return pkt, nil
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
	select {
	case <-p.closed:
		return ErrPollingConnClosed
	case p.out <- pkt:
	}
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
		if _, err := pkt.WriteTo(w); err != nil {
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
			p.in <- &payload.packets[i]
		}
		http.Error(w, "OK", http.StatusOK)
	default:
		http.Error(w, "error", http.StatusMethodNotAllowed)
	}
}

var _ Conn = NewPollingConn(1)

func writeJSONP(w http.ResponseWriter, jsonp string, rd io.Reader) error {
	var buf bytes.Buffer
	w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
	if _, err := buf.ReadFrom(rd); err != nil {
		return err
	}
	tmpl := template.JSEscapeString(buf.String())
	_, err := fmt.Fprintf(w, `___eio[%s]("%s");`, jsonp, tmpl)
	return err
}

func writeXHR(w http.ResponseWriter, rd io.Reader) error {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := io.Copy(w, rd); err != nil {
		return err
	}
	return nil
}
