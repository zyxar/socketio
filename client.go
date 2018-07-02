package engio

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
	*eventHandlers
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
	eventHandlers := newEventHandlers()
	so := &Socket{conn}
	c = &Client{
		Socket:        so,
		eventHandlers: eventHandlers,
		pingInterval:  pingInterval,
		pingTimeout:   pingTimeout,
		closeChan:     closeChan,
		id:            param.SID,
	}

	go func() {
		for {
			select {
			case <-closeChan:
				return
			case <-time.After(pingInterval):
			}
			if err = c.Ping(); err != nil {
				println(err.Error())
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-closeChan:
				return
			default:
			}
			if err := eventHandlers.handle(so); err != nil {
				println(err.Error())
				return
			}
		}
	}()

	return
}

func send(conn Conn, msgType MessageType, pktType PacketType, data []byte) (err error) {
	wc, err := conn.NextWriter(msgType, pktType)
	if err != nil {
		return
	}
	if len(data) > 0 {
		if _, err = wc.Write(data); err != nil {
			wc.Close()
			return
		}
	}
	return wc.Close()
}

func (c *Client) Ping() error {
	return send(c.Conn, MessageTypeString, PacketTypePing, nil)
}

func (c *Client) Close() (err error) {
	c.once.Do(func() {
		close(c.closeChan)
		err = c.Conn.Close()
	})
	return
}

func (c *Client) Id() string {
	return c.id
}
