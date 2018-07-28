package socketio

import (
	"net/http"

	"github.com/zyxar/socketio/engine"
)

// Client is socket.io client
type Client struct {
	engine *engine.Client
	Socket

	onConnect func(string, Socket)
	onError   func(err interface{})
}

// Dial connects to a socket.io server represented by `rawurl` and create Client instance on success.
func Dial(rawurl string, requestHeader http.Header, dialer engine.Dialer, parser Parser, onConnect func(string, Socket)) (c *Client, err error) {
	e, err := engine.Dial(rawurl, requestHeader, dialer)
	if err != nil {
		return
	}
	socket := newClientSocket(e.Socket, parser)
	c = &Client{engine: e, Socket: socket, onConnect: onConnect}
	e.On(engine.EventMessage, engine.Callback(func(_ *engine.Socket, msgType engine.MessageType, data []byte) {
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

	e.On(engine.EventClose, engine.Callback(func(_ *engine.Socket, _ engine.MessageType, _ []byte) {
		socket.Close()
		socket.mutex.Lock()
		for k := range socket.nsp {
			if socket.onDisconnect != nil {
				socket.onDisconnect(k)
			}
		}
		socket.mutex.Unlock()
	}))

	return
}

// Sid returns session id assigned by socket.io server
func (c *Client) Sid() string {
	return c.engine.Sid()
}

// Close closes underlying engine.io transport
func (c *Client) Close() error {
	return c.engine.Close()
}

// OnError registers fn as error callback
func (c *Client) OnError(fn func(interface{})) {
	c.onError = fn
}

// process is the Packet process handle on client side
func (c *Client) process(sock *socket, p *Packet) {
	nsp := sock.creatensp(p.Namespace)

	switch p.Type {
	case PacketTypeConnect:
		if c.onConnect != nil {
			c.onConnect(p.Namespace, sock)
		}
	case PacketTypeDisconnect:
		if sock.onDisconnect != nil {
			sock.onDisconnect(p.Namespace)
		}
	case PacketTypeEvent, PacketTypeBinaryEvent:

		event, data, bin, err := sock.decoder.ParseData(p)
		if err != nil {
			if sock.onError != nil {
				sock.onError(p.Namespace, err)
			}
			return
		}
		if event == "" {
			return
		}
		v, err := nsp.fireEvent(event, data, bin, sock.decoder)
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

	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil {
			_, data, bin, err := sock.decoder.ParseData(p)
			if err != nil {
				if sock.onError != nil {
					sock.onError(p.Namespace, err)
				}
				return
			}
			nsp.fireAck(*p.ID, data, bin, sock.decoder)
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
