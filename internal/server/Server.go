package server

import (
	"errors"
	"io"
	"net"
	"os"
	"serveg/internal/http"
	"serveg/internal/log"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	Conf     *Config
	listener net.Listener
	logger   log.LoggerContext
	wg       sync.WaitGroup
}

func (srv *Server) StartGoroutine() (err error) {
	srv.logger = log.LoggerContext{Name: "server"}
	srv.logger.Info("listening at ", srv.Conf.Addr)
	srv.listener, err = net.Listen("tcp", srv.Conf.Addr)
	if err == nil {
		srv.wg.Go(srv.serve)
	}
	return
}

func (srv *Server) serve() {
	srv.logger.Info("server started")
	for i := int64(0); ; i++ {
		conn, err := srv.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				srv.logger.Info("listener socket closed")
				return
			}
			srv.logger.Warn(err)
			continue
		}
		// TODO: add connection to map. SetDeadline(Now()) in Stop().
		// srv.wg.Go(func() { handleConnection(conn.(*net.TCPConn), i) })
		go handleConnection(conn.(*net.TCPConn), i)
	}
}

func (srv *Server) Stop() {
	srv.logger.Info("stopping server")
	_ = srv.listener.Close()
	srv.wg.Wait()
}

func handleConnection(socket *net.TCPConn, n int64) {
	logger := log.LoggerContext{Name: "conn-" + strconv.FormatInt(n, 10)}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic: ", r)
		}
		_ = socket.Close()
		logger.Info("connection closed")
	}()

	conn := http.NewConnection(socket)
	for {
		err := socket.SetDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			logger.Warn(err)
			return
		}

		req, err := conn.ReadRequest()
		if err != nil {
			logConnError(logger, err)

			if httpErr, ok := errors.AsType[*http.HttpError](err); ok {
				// return error code to client
				logger.Info("sending error response")
				// req is partially parsed, may use fields
				if err = conn.SendErrorResponse(httpErr, req, true); err != nil {
					logger.Warn("can't send error response")
					return
				}
			}
			// close the connection
			logger.Info("lingering close")
			lingeringClose(socket)
			return
		}
		logger.Info(req.Method, " ", req.Path, " ", req.Protocol,
			"\nHeaders: ", req.Headers)

		resBody := "Hi\n" + time.Now().Format(time.DateTime)

		res := conn.CreateResponse(http.StatusOK, req, false)
		res.AttachBodySmallStr(resBody, http.TextPlain)
		logger.Info("sending response: ", res.Status,
			"\nHeaders: ", res.Headers)
		err = conn.SendResponse(res)
		if err != nil {
			logConnError(logger, err)
			return
		}

		// it is set by CreateResponse
		if res.Headers["connection"] == "close" {
			logger.Info("lingering close")
			lingeringClose(socket)
			return
		}
	}
}

func logConnError(logger log.LoggerContext, err error) {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		logger.Info("time out")
	} else if errors.Is(err, io.EOF) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) {

		logger.Info("closed by client")
	} else {
		logger.Warn(err)
	}
}

// lingeringClose: shut down our side, then drain the client's unread data,
// so close() doesn't send RST and destroy the response before it's read.
// Limits: 500 ms, 32 KB
func lingeringClose(socket *net.TCPConn) {
	if socket.CloseWrite() != nil {
		return
	}
	_ = socket.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = io.Copy(io.Discard, io.LimitReader(socket, 32*1024))
}
