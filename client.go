package engineio

import (
	"net/http"
)

type Client struct {
	Conn
}

func Dial(rawurl string, requestHeader http.Header, tr Transport) (*Client, error) {
	conn, err := tr.Dial(rawurl, requestHeader)
	if err != nil {
		return nil, err
	}
	return &Client{conn}, nil
}
