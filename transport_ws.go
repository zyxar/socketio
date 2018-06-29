package engineio

import (
	"io"
	"io/ioutil"
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
	q.Set("EIO", "3")
	q.Set("transport", "websocket")
	u.RawQuery = q.Encode()
	dialer := &websocket.Dialer{ReadBufferSize: t.ReadBufferSize, WriteBufferSize: t.WriteBufferSize}
	c, _, err := dialer.Dial(u.String(), requestHeader)
	if err != nil {
		return nil, err
	}

	return &websocketConn{conn: c}, nil
}

type websocketConn struct {
	conn           *websocket.Conn
	ReceiveTimeout time.Duration
	SendTimeout    time.Duration
}

// LocalAddr returns the local network address.
func (w *websocketConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (w *websocketConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *websocketConn) NextReader() (int, PacketType, io.ReadCloser, error) {
	msgType, reader, err := w.conn.NextReader()
	if err != nil {
		return 0, 0, nil, err
	}

	b := []byte{0}
	if _, err = io.ReadFull(reader, b); err != nil {
		return 0, 0, nil, err
	}

	switch msgType {
	case websocket.TextMessage:
		b[0] -= '0'
	}

	return msgType, PacketType(b[0]), ioutil.NopCloser(reader), nil
}

func (w *websocketConn) NextWriter(msgType int, pt PacketType) (io.WriteCloser, error) {
	wc, err := w.conn.NextWriter(msgType)
	if err != nil {
		return nil, err
	}
	b := []byte{byte(pt)}
	switch msgType {
	case websocket.TextMessage:
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
