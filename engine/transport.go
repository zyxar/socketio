package engine

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// Transport is compositionof Dailer and Acceptor
type Transport interface {
	Dialer
	Acceptor
	Name() string
}

// Dialer dials to remote server and returns a connection instance (client side)
type Dialer interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
}

// Acceptor accepts a connecting requests and creates a connection instance (server side)
type Acceptor interface {
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
}

// Conn is abstraction of bidirectional engine.io connection
type Conn interface {
	PacketReader
	PacketWriter
	io.Closer
	FlushOut() []*Packet
	FlushIn() []*Packet
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Pause() error
	Resume() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	// private
	httpHeader() http.Header
	copyHeaderFrom(conn Conn)
}

func getTransport(name string) Transport {
	switch name {
	case transportWebsocket:
		return WebsocketTransport
	case transportPolling:
		return PollingTransport
	}
	return nil
}

// PacketReader reads data from remote and outputs a Packet when appropriate
type PacketReader interface {
	ReadPacket() (p *Packet, err error)
}

// PacketWriter accepts a Packet and sends to remote
type PacketWriter interface {
	WritePacket(p *Packet) error
}

const (
	transportWebsocket string = "websocket"
	transportPolling   string = "polling"
)

var (
	// ErrPauseNotSupported indicates that a connection does not support PAUSE (e.g. websocket)
	ErrPauseNotSupported = errors.New("transport pause unsupported")
)

// WebsocketTransport is a Transport instance for websocket
var WebsocketTransport Transport = &websocketTransport{}

type websocketTransport struct {
	websocket.Upgrader
}

func (websocketTransport) Name() string {
	return transportWebsocket
}

func (t *websocketTransport) Accept(w http.ResponseWriter, r *http.Request) (Conn, error) {
	if t.Upgrader.CheckOrigin == nil { // allow all connections by default
		t.Upgrader.CheckOrigin = func(_ *http.Request) bool { return true }
	}
	c, err := t.Upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return nil, err
	}
	return &websocketConn{conn: c, header: cloneHTTPHeader(r.Header)}, nil
}

func (t *websocketTransport) Dial(rawurl string, requestHeader http.Header) (Conn, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set(queryEIO, Version)
	q.Set(queryTransport, transportWebsocket)
	u.RawQuery = q.Encode()
	dialer := &websocket.Dialer{
		ReadBufferSize:  t.Upgrader.ReadBufferSize,
		WriteBufferSize: t.Upgrader.WriteBufferSize,
	}
	c, _, err := dialer.Dial(u.String(), requestHeader)
	if err != nil {
		return nil, err
	}
	return &websocketConn{conn: c, header: cloneHTTPHeader(requestHeader)}, nil
}

func copyHeaderFrom(header http.Header, conn Conn) {
	for k, vv := range conn.httpHeader() {
		if _, ok := header[k]; !ok {
			vv2 := make([]string, len(vv))
			copy(vv2, vv)
			header[k] = vv2
		}
	}
}

func cloneHTTPHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

type pollingAcceptor struct{}

func (pollingAcceptor) Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error) {
	return newPollingConn(8, r.Host, r.RemoteAddr, r.Header), nil
}

// PollingAcceptor is an Acceptor instance for polling
var PollingAcceptor Acceptor = &pollingAcceptor{}

type pollingTransport struct{ pollingAcceptor }

func (pollingTransport) Name() string {
	return transportPolling
}

func (pollingTransport) Dial(rawurl string, requestHeader http.Header) (Conn, error) {
	return nil, errors.New("not implemented")
}

// PollingTransport is a Transport instance for polling
var PollingTransport Transport = &pollingTransport{}
