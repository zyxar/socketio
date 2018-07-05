package engio

import (
	"io"
	"net/http"
)

type Transport interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
}

type Conn interface {
	PacketReader
	PacketWriter
	io.Closer
}

func getTransport(tr string) Transport {
	switch tr {
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
