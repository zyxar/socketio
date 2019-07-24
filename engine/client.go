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
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		tick := time.NewTicker(pingInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			if err = ß.Emit(EventPing, MessageTypeString, nil); err != nil {
				select {
				case <-ctx.Done():
				default:
					log.Println("engine.io ping:", err.Error())
				}
				return
			}
		}
	}()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
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
			if err = c.handle(p); err != nil {
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

func (c *Client) handle(p *Packet) (err error) {
	switch p.pktType {
	case PacketTypeOpen:
	case PacketTypeClose:
		c.fire(c.Socket, EventClose, p.msgType, p.data)
		return c.Socket.Close()
	case PacketTypePing:
	case PacketTypePong:
	case PacketTypeMessage:
		c.fire(c.Socket, EventMessage, p.msgType, p.data)
	case PacketTypeUpgrade:
	case PacketTypeNoop:
	default:
		return ErrInvalidPayload
	}
	return
}

// Close closes underlying connection and signals stop for background workers
func (c *Client) Close() (err error) {
	c.cancel()
	err = c.Socket.Close()
	c.wg.Wait()
	return
}
