package socketio

import (
	"net/http"

	"github.com/zyxar/socketio/engine"
)

type Client struct {
	engine *engine.Client
	Socket
}

func Dial(rawurl string, requestHeader http.Header, dialer engine.Dialer, parser Parser) (c *Client, err error) {
	e, err := engine.Dial(rawurl, requestHeader, dialer)
	if err != nil {
		return
	}
	socket := newClientSocket(e.Socket, parser)
	e.Socket.On(engine.EventMessage, engine.Callback(func(msgType engine.MessageType, data []byte) {
		switch msgType {
		case engine.MessageTypeString:
		case engine.MessageTypeBinary:
		default:
			return
		}
		if err := socket.decoder.Add(msgType, data); err != nil {
			if socket.onError != nil {
				socket.onError(err)
			}
		}
		if p := socket.yield(); p != nil {
			socket.process(p)
		}
	}))

	e.Socket.On(engine.EventClose, engine.Callback(func(_ engine.MessageType, _ []byte) {
		socket.Close()
	}))

	c = &Client{engine: e, Socket: socket}
	return
}

func (c *Client) Id() string {
	return c.engine.Id()
}

func (c *Client) Close() error {
	return c.engine.Close()
}
