package engio

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
}

type Dialer interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
}

type Acceptor interface {
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
	Transport() string
}

type Conn interface {
	PacketReader
	PacketWriter
	io.Closer
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

func (websocketTransport) Transport() string {
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
	q.Set(queryTransport, "websocket")
	u.RawQuery = q.Encode()
	dialer := &websocket.Dialer{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, _, err := dialer.Dial(u.String(), requestHeader)
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
}

// polling

type pollingAcceptor struct {
}

func (p *pollingAcceptor) Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error) {
	return NewPollingConn(8), nil
}

var PollingAcceptor Acceptor = &pollingAcceptor{}

func (pollingAcceptor) Transport() string {
	return transportPolling
}
