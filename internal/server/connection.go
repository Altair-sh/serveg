package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"serveg/internal/html"
	"serveg/internal/http"
	"serveg/internal/log"
	"strconv"
	"syscall"
	"time"
)

type connection struct {
	srv    *Server
	n      int64
	socket *net.TCPConn
	log    log.LoggerContext
	http   *http.Connection
	req    *http.Request
}

func newConnection(srv *Server, n int64, socket *net.TCPConn) *connection {
	return &connection{
		srv:    srv,
		n:      n,
		socket: socket,
		log:    log.LoggerContext{Name: "conn-" + strconv.FormatInt(n, 10)},
		http:   http.NewConnection(socket, srv.timeout),
	}
}
func (conn *connection) handle() {
	defer func() {
		if r := recover(); r != nil {
			conn.log.Error("panic: ", r)
		}
		_ = conn.socket.Close()
		conn.log.Info("connection closed")
	}()

	for {
		err := conn.socket.SetDeadline(time.Now().Add(conn.srv.timeout))
		if err != nil {
			conn.log.Warn(err)
			return
		}

		var res *http.Response
		conn.req, err = conn.http.ReadRequest()
		if err != nil {
			// invalid request
			if httpErr, ok := errors.AsType[*http.HttpError](err); ok {
				conn.log.Warn(err)
				res = conn.http.CreateErrorResponse(conn.req, true, httpErr)
			} else {
				conn.logError(err)
				conn.lingeringClose()
				return
			}
		} else {
			// valid request
			conn.log.Info(conn.req.Method, " ", conn.req.Url, " ", conn.req.Protocol)
			conn.log.Trace("request headers: ", conn.req.Headers)
			switch conn.req.Method {
			case "GET", "HEAD":
				res = conn.serveFileOrDir()
			case "PUT", "OPTIONS":
				res = conn.handleUpload()
			default:
				res = conn.createErrorResponse(false, http.StatusMethodNotAllowed, conn.req.Method)
			}
		}

		conn.log.Info("sending response: ", res.Status,
			" content-length: ", res.Headers["content-length"])
		conn.log.Trace("response headers: ", res.Headers)
		err = conn.http.SendResponse(res)
		if err != nil {
			conn.logError(err)
			return
		}

		// it is set by CreateResponse
		if res.Headers["connection"] == "close" {
			conn.lingeringClose()
			return
		}
	}
}

func (conn *connection) logError(err error) {
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
func (conn *connection) lingeringClose() {
	conn.log.Info("lingering close")
	if conn.socket.CloseWrite() != nil {
		return
	}
	_ = conn.socket.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = io.Copy(io.Discard, io.LimitReader(conn.socket, 32*1024))
}

func (conn *connection) createErrorResponse(
	wantsClose bool, status http.StatusCode, msg string,
) *http.Response {
	return conn.http.CreateErrorResponse(conn.req, wantsClose, http.NewHttpError(status, msg))
}

func (conn *connection) serveFileOrDir() *http.Response {
	path, share, urlStatus := conn.srv.Conf.Share.MapUrl(conn.req.Url)
	// TODO implement authorization
	_ = share
	switch urlStatus {
	case UrlFile:
		return conn.serveFile(path)
	case UrlDir:
		return conn.serveDirListing(path)
	case UrlNotFound:
		return conn.createErrorResponse(false, http.StatusNotFound, path)
	case UrlForbidden:
		return conn.createErrorResponse(false, http.StatusForbidden, path)
	case UrlIncorrect:
		return conn.createErrorResponse(false, http.StatusBadRequest,
			"invalid url '"+conn.req.Url+"'")
	default:
		panic(fmt.Sprintf("unsupported UrlStatus: %d", int32(urlStatus)))
	}
}

func (conn *connection) serveDirListing(dirPath string) *http.Response {
	htmlBytes, err := html.GenDirListingHtml(&conn.log, dirPath,
		conn.req.Url, conn.srv.Conf.HtmlTitle)
	if err != nil {
		return conn.createErrorResponse(false, http.StatusInternalServerError, err.Error())
	}

	return conn.http.CreateResponse(http.StatusOK, conn.req, false).
		WithBodyBuf(htmlBytes, http.TextHtml)
}

func (conn *connection) serveFile(filePath string) *http.Response {
	f, err := os.Open(filePath)
	if err != nil {
		return conn.createErrorResponse(false, http.StatusInternalServerError, err.Error())
	}
	var fileOwnershipTransferred bool
	defer func() {
		if !fileOwnershipTransferred {
			_ = f.Close()
		}
	}()

	finf, err := f.Stat()
	if err != nil {
		return conn.createErrorResponse(false, http.StatusInternalServerError, err.Error())
	}
	size := finf.Size()

	// time is received with second precision
	// if not truncate, mtime will always be > ifModifiedSince
	mtime := finf.ModTime().UTC().Truncate(time.Second)

	ifModifiedSince, errTime := time.Parse(http.TimeFormat, conn.req.Headers["if-modified-since"])
	notModified := errTime == nil && !mtime.After(ifModifiedSince)
	if notModified {
		return conn.http.CreateResponse(http.StatusNotModified, conn.req, false)
	}

	res := conn.http.CreateResponse(http.StatusOK, conn.req, false)
	fileName := path.Base(filePath)

	if conn.req.Method != "HEAD" {
		// file will be closed by SendResponse()
		fileOwnershipTransferred = true
		res.WithBodyReader(f, size, http.GetMimeType(fileName))
	}
	// Tell browser to not use its own caching, remember last-modified
	// On next request browser will send if-modified-since and let server check if file changed.
	res.Headers["cache-control"] = "no-cache"
	res.Headers["last-modified"] = mtime.Format(http.TimeFormat)
	res.Headers["content-disposition"] = http.ContentDisposition("inline", fileName)
	//TODO: accept "range" header and return StatusPartialContent
	// res.Headers["accept-ranges"] = "bytes"

	return res
}

// TODO: implement request body and then file uploading
func (conn *connection) handleUpload() *http.Response {
	// TODO: use os.OpenFile(filePath + ".upload", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	// if such file exists, send response like
	// "somebody is uploading such file right now"
	// TODO: defer { if !renamed { os.Remove(tmp) } }
	panic("unimplemented")
}
