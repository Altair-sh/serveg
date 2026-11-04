package server

import (
	"net"
	"serveg/internal/http"
	"serveg/internal/log"
	"strconv"
)

type TcpConnection struct {
	srv    *Server
	socket *net.TCPConn
	log    log.LoggerContext
	http   *http.Connection
}

func NewTcpConnection(srv *Server, socket *net.TCPConn, n int64) *TcpConnection {
	conn := &TcpConnection{
		srv:    srv,
		socket: socket,
		log:    log.LoggerContext{Name: "conn-" + strconv.FormatInt(n, 10)},
	}
	conn.http = http.NewConnection(socket, &conn.log, conn.handleHttpRequest)
	return conn
}

func (conn *TcpConnection) Start() {
	conn.http.Start()
}

func (conn *TcpConnection) handleHttpRequest(req *http.Request) (res *http.Response) {
	switch req.Method {
	case "GET", "HEAD":
		res = conn.serveFileOrDir(req)
	case "PUT", "OPTIONS":
		res = conn.handleUpload(req)
	default:
		res = conn.createErrorResponse(req, false, http.StatusMethodNotAllowed, req.Method)
		res.Headers["allow"] = "GET, HEAD, PUT, OPTIONS"
	}
	return
}

func (conn *TcpConnection) createErrorResponse(
	req *http.Request, wantsClose bool, status http.StatusCode, msg string,
) *http.Response {
	return conn.http.CreateErrorResponse(req, wantsClose, http.NewHttpError(status, msg))
}
