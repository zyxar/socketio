package engine

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type Transport interface {
	Dialer
	Acceptor
	Name() string
}

type Dialer interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
}

type Acceptor interface {
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
}

type Conn interface {
	PacketReader
	PacketWriter
	io.Closer
	FlushOut() []*Packet
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Pause() error
	Resume() error
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

func getAcceptor(name string) Acceptor {
	switch name {
	case transportWebsocket:
		return WebsocketTransport
	case transportPolling:
		return PollingAcceptor
	}
	return nil
}

func getDialer(name string) Dialer {
	switch name {
	case transportWebsocket:
		return WebsocketTransport
	case transportPolling:
	}
	return nil
}

type PacketReader interface {
	ReadPacket() (p *Packet, err error)
}

type PacketWriter interface {
	WritePacket(p *Packet) error
}

const (
	transportWebsocket string = "websocket"
	transportPolling   string = "polling"
)

var (
	ErrPauseNotSupported = errors.New("transport pause unsupported")
)

// websocket

var WebsocketTransport Transport = &websocketTransport{}

type websocketTransport struct {
	ReadBufferSize  int
	WriteBufferSize int
}

func (websocketTransport) Name() string {
	return transportWebsocket
}

func (t *websocketTransport) Accept(w http.ResponseWriter, r *http.Request) (Conn, error) {
	upgrader := &websocket.Upgrader{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
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
	dialer := &websocket.Dialer{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, _, err := dialer.Dial(u.String(), requestHeader)
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
}

// polling

type pollingAcceptor struct{}

func (p *pollingAcceptor) Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error) {
	return NewPollingConn(8), nil
}

var PollingAcceptor Acceptor = &pollingAcceptor{}

type pollingTransport struct{ *pollingAcceptor }

func (pollingTransport) Name() string {
	return transportPolling
}

func (pollingTransport) Dial(rawurl string, requestHeader http.Header) (Conn, error) {
	return nil, errors.New("not implemented")
}

var PollingTransport Transport = &pollingTransport{&pollingAcceptor{}}
