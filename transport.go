package engineio

import (
	"net/http"
)

type Transport interface {
	Dial(rawurl string, requestHeader http.Header) (conn Conn, err error)
	Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error)
}

type Conn interface {
	ReadMessage() (message []byte, err error)
	WriteMessage(message []byte) error
	Close() error
}
