package engineio

import (
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
	PingInterval   time.Duration
	PingTimeout    time.Duration
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

func (w *websocketConn) ReadMessage() (message []byte, err error) {
	w.conn.SetReadDeadline(time.Now().Add(w.ReceiveTimeout))
	msgType, reader, err := w.conn.NextReader()
	if err != nil {
		return
	}

	//support only text messages exchange
	if msgType != websocket.TextMessage {
		// return "", ErrorBinaryMessage
	}
	message, err = ioutil.ReadAll(reader)
	if err != nil {
		return
	}

	//empty messages are not allowed
	// if len(text) == 0 {
	// 	return "", ErrorPacketWrong
	// }

	return
}

func (w *websocketConn) WriteMessage(message []byte) error {
	w.conn.SetWriteDeadline(time.Now().Add(w.SendTimeout))
	wc, err := w.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	if _, err = wc.Write(message); err != nil {
		return err
	}
	return wc.Close()
}

func (w *websocketConn) Close() error {
	return w.conn.Close()
}
