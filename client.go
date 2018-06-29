package engineio

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrInvalidPacket    = errors.New("invalid packet")
	ErrUnexpectedPacket = errors.New("unexpected packet")
)

type Client struct {
	Conn
	param Parameters
}

func Dial(rawurl string, requestHeader http.Header, tr Transport) (c *Client, err error) {
	conn, err := tr.Dial(rawurl, requestHeader)
	if err != nil {
		return
	}
	_, pt, rc, err := conn.NextReader()
	if err != nil {
		return
	}
	if pt != PacketTypeOpen {
		err = ErrUnexpectedPacket
		return
	}
	var param Parameters
	if err = json.NewDecoder(rc).Decode(&param); err != nil {
		return
	}
	c = &Client{Conn: conn, param: param}
	return
}
