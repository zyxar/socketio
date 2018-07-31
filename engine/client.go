package engine

import (
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
	closeChan chan struct{}
	once      sync.Once
}

// Dial connects to a engine.io server represented by `rawurl` and create Client instance on success.
func Dial(rawurl string, requestHeader http.Header, dialer Dialer) (c *Client, err error) {
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

	closeChan := make(chan struct{}, 1)
	ß := newSocket(conn, pingTimeout, pingTimeout, param.SID)
	c = &Client{
		Socket:        ß,
		eventHandlers: newEventHandlers(),
		closeChan:     closeChan,
	}

	go func() {
		var err error
		for {
			select {
			case <-closeChan:
				return
			case <-time.After(pingInterval):
			}
			if err = ß.Emit(EventPing, MessageTypeString, nil); err != nil {
				log.Println("emit:", err.Error())
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
			case <-closeChan:
				return
			default:
			}
			if p, err = ß.Read(); err != nil {
				log.Println("read:", err.Error())
				return
			}
			if err = c.handle(ß, p); err != nil {
				log.Println("handle:", err.Error())
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
		close(c.closeChan)
		err = c.Conn.Close()
	})
	return
}
