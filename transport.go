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
	NextReader() (MessageType, PacketType, io.ReadCloser, error)
	NextWriter(MessageType, PacketType) (io.WriteCloser, error)
	Close() error
}
