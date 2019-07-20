package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrPollingConnClosed implies connection closed; fatal.
	ErrPollingConnClosed = errors.New("polling connection closed")
	// ErrPollingConnReadTimeout implies connection read timeout; fatal.
	ErrPollingConnReadTimeout = errors.New("polling connection read timeout")
	// ErrPollingConnWriteTimeout implies connection write timeout; fatal.
	ErrPollingConnWriteTimeout = errors.New("polling connection write timeout")
	// ErrPollingConnPaused implies connection paused; temperary.
	ErrPollingConnPaused = errors.New("polling connection paused")
	// ErrPollingRequestCanceled implies request canceled; temperary.
	ErrPollingRequestCanceled = errors.New("polling request canceled")
)

type pollingConn struct {
	in            chan *Packet
	out           chan *Packet
	closed        chan struct{}
	once          sync.Once
	readDeadline  atomic.Value
	writeDeadline atomic.Value
	paused        atomic.Value
	localAddr     netAddr
	remoteAddr    netAddr
	header        http.Header
}

func (p *pollingConn) Close() error {
	p.once.Do(func() {
		close(p.closed)
	})
	return nil
}

// newPollingConn creates an instance of polling connection
func newPollingConn(bufSize int, localAddr, remoteAddr string, header http.Header) *pollingConn {
	p := &pollingConn{
		in:         make(chan *Packet, bufSize),
		out:        make(chan *Packet, bufSize),
		closed:     make(chan struct{}),
		localAddr:  netAddr{addr: localAddr},
		remoteAddr: netAddr{addr: remoteAddr},
		header:     cloneHTTPHeader(header),
	}
	p.paused.Store(make(chan struct{}))
	return p
}

// LocalAddr returns the local network address.
func (p *pollingConn) LocalAddr() net.Addr { return p.localAddr }

// RemoteAddr returns the remote network address.
func (p *pollingConn) RemoteAddr() net.Addr { return p.remoteAddr }

// GetHeader returns the value in http header from client request specified by `key`
func (p *pollingConn) GetHeader(key string) string { return p.header.Get(key) }

func (p *pollingConn) ReadPacket() (*Packet, error) {
	if p.isClosed() {
		return nil, ErrPollingConnClosed
	}
	var timer <-chan time.Time
	t := p.readDeadline.Load()
	if t != nil {
		deadline := t.(time.Time)
		if !deadline.IsZero() {
			timeout := time.Until(deadline)
			if timeout > 0 {
				timer = time.After(timeout)
			} else {
				return nil, ErrPollingConnReadTimeout
			}
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

func (p *pollingConn) FlushOut() (packets []*Packet) {
	for {
		select {
		case pkt := <-p.out:
			packets = append(packets, pkt)
		default:
			return
		}
	}
}

func (p *pollingConn) FlushIn() (packets []*Packet) {
	for {
		select {
		case pkt := <-p.in:
			packets = append(packets, pkt)
		default:
			return
		}
	}
}

func (p *pollingConn) ReadPacketOut(ctx context.Context) (*Packet, error) {
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
	case <-ctx.Done():
		return nil, ErrPollingRequestCanceled
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
		if !deadline.IsZero() {
			timeout := time.Until(deadline)
			if timeout > 0 {
				timer = time.After(timeout)
			} else {
				return ErrPollingConnWriteTimeout
			}
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
	select {
	case <-p.closed:
		return ErrPollingConnClosed
	case <-p.pauseChan():
		return ErrPollingConnPaused
	default:
	}
	p.readDeadline.Store(t)
	return nil
}

func (p *pollingConn) SetWriteDeadline(t time.Time) error {
	select {
	case <-p.closed:
		return ErrPollingConnClosed
	case <-p.pauseChan():
		return ErrPollingConnPaused
	default:
	}
	p.writeDeadline.Store(t)
	return nil
}

func (p *pollingConn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.isClosed() {
		http.Error(w, ErrPollingConnClosed.Error(), http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		pkt, err := p.ReadPacketOut(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		q := r.URL.Query()
		b64 := q.Get(queryBase64)
		if jsonp := q.Get(queryJSONP); jsonp != "" {
			err = writeJSONP(w, jsonp, pkt)
		} else if b64 == "1" {
			err = writeXHR(w, pkt)
		} else {
			err = writeXHR2(w, pkt.packet2())
		}
		if err != nil {
			log.Println("polling:", err.Error())
		}
	case "POST":
		var payload Payload
		mediatype, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch mediatype {
		case "application/octet-stream":
			payload.xhr2 = true
		case "text/plain":
			if strings.ToLower(params["charset"]) != "utf-8" {
				http.Error(w, "invalid charset", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "invalid media type", http.StatusBadRequest)
			return
		}
		_, err = payload.ReadFrom(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for i := range payload.packets {
			select {
			case <-p.closed:
				http.Error(w, "closed", http.StatusNotFound)
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

type netAddr struct {
	addr string
}

func (netAddr) Network() string {
	return "tcp"
}

func (n netAddr) String() string {
	return n.addr
}

var _ Conn = newPollingConn(1, "", "", nil)

func writeJSONP(w http.ResponseWriter, jsonp string, wt io.WriterTo) error {
	var buf bytes.Buffer
	w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
	if _, err := wt.WriteTo(&buf); err != nil {
		return err
	}
	s := buf.String()
	buf.Reset()
	err := json.NewEncoder(&buf).Encode(s)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, `___eio[%s]("%s");`, jsonp, buf.String())
	return err
}

func writeXHR(w http.ResponseWriter, wt io.WriterTo) error {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	if _, err := wt.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func writeXHR2(w http.ResponseWriter, wt io.WriterTo) error {
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := wt.WriteTo(w); err != nil {
		return err
	}
	return nil
}
