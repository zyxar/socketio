package engio

import (
	"io"
	"net/http"
	"time"
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
}

type Conn interface {
	PacketReader
	PacketWriter
	io.Closer
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

func getTransport(name string) Transport {
	switch name {
	case "websocket":
		return WebsocketTransport
	case "polling":
	}
	return nil
}

func getAcceptor(name string) Acceptor {
	switch name {
	case "websocket":
		return WebsocketTransport
	case "polling":
		return PollingAcceptor
	}
	return nil
}

func getDialer(name string) Dialer {
	switch name {
	case "websocket":
		return WebsocketTransport
	case "polling":
	}
	return nil
}

type PacketReader interface {
	ReadPacket() (p *Packet, err error)
}

type PacketWriter interface {
	WritePacket(p *Packet) error
}
