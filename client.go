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
			c.process(socket, p)
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

func (*Client) process(s *socket, p *Packet) {
	switch p.Type {
	case PacketTypeConnect:
		s.fire(p.Namespace, "connect", nil, nil)
	case PacketTypeDisconnect:
		s.mutex.Lock()
		delete(s.nsp, p.Namespace)
		s.mutex.Unlock()
	case PacketTypeEvent, PacketTypeBinaryEvent:
		if p.event != nil {
			v, err := s.fire(p.Namespace, p.event.name, p.event.data, p.buffer)
			if err != nil {
				if s.onError != nil {
					s.onError(err)
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
				if err = s.ack(p); err != nil {
					if s.onError != nil {
						s.onError(err)
					}
				}
			}
		}
	case PacketTypeAck, PacketTypeBinaryAck:
		if p.ID != nil && p.event != nil {
			s.namespace(p.Namespace).onAck(*p.ID, p.event.data, p.buffer)
		}
	case PacketTypeError:
	default:
		if s.onError != nil {
			s.onError(ErrUnknownPacket)
		}
	}
}
