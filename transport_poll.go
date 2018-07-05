package engio

import (
	"net/http"
)

type pollingAcceptor struct {
}

func (p *pollingAcceptor) Accept(w http.ResponseWriter, r *http.Request) (conn Conn, err error) {
	return NewPollingConn(8), nil
}

var PollingAcceptor Acceptor = &pollingAcceptor{}
