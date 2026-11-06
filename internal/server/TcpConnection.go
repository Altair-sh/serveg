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

func (conn *TcpConnection) handleHttpRequest(req *http.Request) *http.Response {
	switch req.Method {
	case "GET", "HEAD":
		return conn.GET_HEAD(req)
	case "PUT", "OPTIONS":
		return conn.PUT_OPTIONS(req)
	default:
		return conn.responseMethodNotAllowed(req, "GET, HEAD, PUT, OPTIONS")
	}
}

func (conn *TcpConnection) createErrorResponse(
	req *http.Request, wantsClose bool, status http.StatusCode, msg string,
) *http.Response {
	return conn.http.CreateErrorResponse(req, wantsClose, http.NewHttpError(status, msg))
}

func (conn *TcpConnection) responseMethodNotAllowed(req *http.Request, allowedMethods string,
) *http.Response {
	return conn.createErrorResponse(req, false,
		http.StatusMethodNotAllowed, req.Method).
		WithHeader("allow", allowedMethods)
}
