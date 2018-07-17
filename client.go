package socketio

import (
	"net/http"

	"github.com/zyxar/socketio/engine"
)

type Client struct {
	engine *engine.Client
	Socket

	onConnect func(Socket)
	onError   func(err interface{})
}

func Dial(rawurl string, requestHeader http.Header, dialer engine.Dialer, parser Parser, onConnect func(Socket)) (c *Client, err error) {
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
			if c.onError != nil {
				c.onError(err)
			}
		}
		if p := socket.yield(); p != nil {
			c.process(socket, p)
		}
	}))

	e.Socket.On(engine.EventClose, engine.Callback(func(_ engine.MessageType, _ []byte) {
		socket.Close()
	}))

	c = &Client{engine: e, Socket: socket, onConnect: onConnect}
	return
}

func (c *Client) Id() string {
	return c.engine.Id()
}

func (c *Client) Close() error {
	return c.engine.Close()
}

func (c *Client) OnError(fn func(interface{})) {
	c.onError = fn
}

func (c *Client) process(sock *socket, p *Packet) {
	switch p.Type {
	case PacketTypeConnect:
		if c.onConnect != nil {
			c.onConnect(sock)
		}
	case PacketTypeDisconnect:
		sock.mutex.Lock()
		delete(sock.nsp, p.Namespace)
		sock.mutex.Unlock()
	case PacketTypeEvent, PacketTypeBinaryEvent:
		if p.event != nil {
			v, err := sock.fire(p.Namespace, p.event.name, p.event.data, p.buffer)
			if err != nil {
				if c.onError != nil {
					c.onError(err)
				}
				return
			}
			if p.ID != nil {
				p.Data = nil
				if v != nil {
					d := make([]interface{}, len(v))
					for i := range d {
						d[i] = v[i].Interface()
					}
					p.Data = d
				}
				if err = sock.ack(p); err != nil {
					if c.onError != nil {
						c.onError(err)
					}
				}
			}
		}
	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil && p.event != nil {
			sock.namespace(p.Namespace).onAck(*p.ID, p.event.data, p.buffer)
		}
	case PacketTypeError:
		if c.onError != nil {
			c.onError(p.Data)
		}
	default:
		if c.onError != nil {
			c.onError(ErrUnknownPacket)
		}
	}
}
