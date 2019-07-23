package engine

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// Client is engine.io client
type Client struct {
	*Socket
	*eventHandlers
	once   sync.Once
	cancel context.CancelFunc
}

// Dial connects to a engine.io server represented by `rawurl` and create Client instance on success.
func Dial(ctx context.Context, rawurl string, requestHeader http.Header, dialer Dialer) (c *Client, err error) {
	conn, err := dialer.Dial(rawurl, requestHeader)
	if err != nil {
		return
	}
	p, err := conn.ReadPacket()
	if err != nil {
		return
	}
	if p.pktType != PacketTypeOpen {
		err = ErrInvalidPayload
		return
	}
	var param Parameters
	if err = json.Unmarshal(p.data, &param); err != nil {
		return
	}
	pingInterval := time.Duration(param.PingInterval) * time.Millisecond
	pingTimeout := time.Duration(param.PingTimeout) * time.Millisecond

	ctx, cancel := context.WithCancel(ctx)
	ß := newSocket(ctx, conn, pingInterval+pingTimeout, pingTimeout, param.SID)
	c = &Client{
		Socket:        ß,
		eventHandlers: newEventHandlers(),
		cancel:        cancel,
	}

	go func() {
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pingInterval):
			}
			if err = ß.Emit(EventPing, MessageTypeString, nil); err != nil {
				select {
				case <-ctx.Done():
				default:
					log.Println("engine.io emit:", err.Error())
				}
				return
			}
		}
	}()
	go func() {
		defer ß.Close()
		var p *Packet
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if p, err = ß.Read(); err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				log.Println("engine.io read:", err.Error())
				return
			}
			if err = c.handle(ß, p); err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				log.Println("engine.io handle:", err.Error())
			}
		}
	}()

	return
}

func (c *Client) handle(ß *Socket, p *Packet) (err error) {
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		c.fire(ß, EventClose, p.msgType, p.data)
		return ß.Close()
	case PacketTypePing:
	case PacketTypePong:
	case PacketTypeMessage:
		c.fire(ß, EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
	case PacketTypeNoop:
	default:
		return ErrInvalidPayload
	}
	return
}

// Close closes underlying connection and signals stop for background workers
func (c *Client) Close() (err error) {
	c.once.Do(func() {
		c.cancel()
		err = c.Conn.Close()
	})
	return
}
