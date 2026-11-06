package http

import (
	"bufio"
	"errors"
	"io"
	"net"
	"os"
	"serveg/internal/log"
	sh "serveg/internal/string_helper"
	"syscall"
	"time"
)

const maxHeaderBytes = 16 * 1024
const initialWriteBufCap = 512

type mapStrStr map[string]string
type HeadersMap map[string]string
type ConnCallback func(req *Request) *Response

type Connection struct {
	Timeout time.Duration

	log      *log.LoggerContext
	callback ConnCallback
	socket   *net.TCPConn
	r        *bufio.Reader
	wbuf     sh.StringBuilder

	// allocated in Connection once
	// hidden to prevent using before ReadRequest/SendResponse
	_req Request  // reset by ReadRequest
	_res Response // reset by CreateResponse and ReadRequest
}

const defaultTimeout = 30 * time.Second

func NewConnection(
	socket *net.TCPConn,
	logger *log.LoggerContext,
	callback ConnCallback,
) *Connection {
	r := bufio.NewReaderSize(socket, maxHeaderBytes)
	return &Connection{
		Timeout:  defaultTimeout,
		log:      logger,
		callback: callback,
		socket:   socket,
		r:        r,
		wbuf:     sh.StringBuilder{Buf: make([]byte, 0, initialWriteBufCap)},
		_req: Request{
			QueryParams: make(mapStrStr),
			Headers:     make(HeadersMap),
			// body is read from the same buffered reader as headers
			bodyReader: io.LimitedReader{R: r},
		},
		_res: Response{
			Headers: make(HeadersMap),
		},
	}
}

func (conn *Connection) Start() {
	defer func() {
		if r := recover(); r != nil {
			conn.log.Error("panic: ", r)
		}
		_ = conn.socket.Close()
		conn.log.Info("connection closed")
	}()

	for {
		err := conn.socket.SetDeadline(time.Now().Add(conn.Timeout))
		if err != nil {
			conn.log.Warn(err)
			return
		}

		var res *Response
		var req *Request
		req, err = conn.ReadRequest()
		if err != nil {
			// invalid request
			if httpErr, ok := errors.AsType[*HttpError](err); ok {
				conn.log.Warn(err)
				res = conn.CreateErrorResponse(req, true, httpErr)
			} else {
				conn.logError(err)
				conn.lingeringClose()
				return
			}
		} else {
			// valid request
			conn.log.Info(req.Method, " ", req.Url, " ", req.Protocol)
			conn.log.Trace("request headers: ", HeadersToString(req.Headers, SecretHeaders))
			res = conn.callback(req)
			if !res.CloseConnection && req.HasUnreadBody() {
				res = conn.CreateErrorResponse(req, true, ErrHasUnreadBody)
			}
		}

		conn.log.Info("sending response: ", res.Status,
			" content-length: ", res.Headers["content-length"])
		conn.log.Trace("response headers: ", HeadersToString(res.Headers, SecretHeaders))
		err = conn.sendResponse(res)
		if err != nil {
			conn.logError(err)
			return
		}

		// it is set by CreateResponse
		if res.CloseConnection {
			conn.lingeringClose()
			return
		}
	}
}

func (conn *Connection) logError(err error) {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		conn.log.Info("time out")
	} else if errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) {

		conn.log.Info("closed by client")
	} else {
		conn.log.Error(err)
	}
}

// lingeringClose: shut down our side, then drain the client's unread data,
// so close() doesn't send RST and destroy the response before it's read.
// Limits: 500 ms, 32 KB
func (conn *Connection) lingeringClose() {
	conn.log.Info("lingering close")
	if conn.socket.CloseWrite() != nil {
		return
	}
	_ = conn.socket.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = io.Copy(io.Discard, io.LimitReader(conn.socket, 32*1024))
}
