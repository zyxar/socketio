package engio

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var WebsocketTransport Transport = &websocketTransport{}

type websocketTransport struct {
	ReadBufferSize  int
	WriteBufferSize int
}

func (websocketTransport) Transport() string {
	return transportWebsocket
}

func (t *websocketTransport) Accept(w http.ResponseWriter, r *http.Request) (Conn, error) {
	upgrader := &websocket.Upgrader{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, err := upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
}

func (t *websocketTransport) Dial(rawurl string, requestHeader http.Header) (Conn, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set(queryEIO, Version)
	q.Set(queryTransport, "websocket")
	u.RawQuery = q.Encode()
	dialer := &websocket.Dialer{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, _, err := dialer.Dial(u.String(), requestHeader)
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
}

type websocketConn struct {
	conn *websocket.Conn
}

// LocalAddr returns the local network address.
func (w *websocketConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (w *websocketConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *websocketConn) nextWriter(msgType MessageType, pt PacketType) (io.WriteCloser, error) {
	var m int
	switch msgType {
	case MessageTypeString:
		m = websocket.TextMessage
	case MessageTypeBinary:
		m = websocket.BinaryMessage
	default:
		return nil, ErrInvalidMessage
	}

	wc, err := w.conn.NextWriter(m)
	if err != nil {
		return nil, err
	}
	b := []byte{byte(pt)}
	switch msgType {
	case MessageTypeString:
		b[0] += '0'
	}
	if _, err := wc.Write(b); err != nil {
		wc.Close()
		return nil, err
	}
	return wc, nil
}

func (w *websocketConn) Close() error {
	return w.conn.Close()
}

func (w *websocketConn) WritePacket(p *Packet) error {
	wc, err := w.nextWriter(p.msgType, p.pktType)
	if err != nil {
		return err
	}
	if len(p.data) > 0 {
		if _, err = wc.Write(p.data); err != nil {
			wc.Close()
			return err
		}
	}
	return wc.Close()
}

func (w *websocketConn) ReadPacket() (p *Packet, err error) {
	msgType, reader, err := w.conn.NextReader()
	if err != nil {
		return nil, err
	}

	b := []byte{0}
	if _, err = io.ReadFull(reader, b); err != nil {
		return nil, err
	}
	switch msgType {
	case websocket.TextMessage:
		b[0] -= '0'
		p = &Packet{MessageTypeString, PacketType(b[0]), nil}
	case websocket.BinaryMessage:
		p = &Packet{MessageTypeBinary, PacketType(b[0]), nil}
	default:
		return nil, ErrInvalidMessage
	}

	var buffer bytes.Buffer
	if _, err = buffer.ReadFrom(reader); err != nil {
		return
	}
	p.data = buffer.Bytes()
	return
}

func (w *websocketConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *websocketConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

func (*websocketConn) Pause() error  { return ErrPauseNotSupported }
func (*websocketConn) Resume() error { return nil }
