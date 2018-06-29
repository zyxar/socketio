package engineio

import (
	"io"
	"net/http"
)

type Transport interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
}

type Conn interface {
	NextReader() (int, PacketType, io.ReadCloser, error)
	NextWriter(int, PacketType) (io.WriteCloser, error)
	Close() error
}
