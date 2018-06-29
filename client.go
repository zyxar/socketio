package engineio

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	ErrInvalidMessage   = errors.New("invalid message")
	ErrUnexpectedPacket = errors.New("unexpected packet")
)

type Client struct {
	*Socket
	id           string
	pingInterval time.Duration
	pingTimeout  time.Duration
	closeChan    chan struct{}
	once         sync.Once
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
	pingInterval := time.Duration(param.PingInterval) * time.Millisecond
	pingTimeout := time.Duration(param.PingTimeout) * time.Millisecond

	closeChan := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-closeChan:
				return
			case <-time.After(pingInterval):
			}
			wc, err := conn.NextWriter(MessageTypeString, PacketTypePing)
			if err != nil {
				return
			}
			if err = wc.Close(); err != nil {
				return
			}
		}
	}()

	c = &Client{
		Socket:       &Socket{conn},
		pingInterval: pingInterval,
		pingTimeout:  pingTimeout,
		closeChan:    closeChan,
		id:           param.SID,
	}
	return
}

func (c *Client) Close() (err error) {
	c.once.Do(func() {
		close(c.closeChan)
		err = c.Conn.Close()
	})
	return
}

func (c *Client) On(event string, callback interface{}) {

}

func (c *Client) Id() string {
	return c.id
}
