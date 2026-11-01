package http

import (
	"errors"
	"io"
	"net"
	sh "serveg/internal/string_helper"
	"strconv"
	"strings"
	"time"
)

type responseBodyType uint8

const (
	responseBodyEmpty = responseBodyType(iota)
	responseBodySmall
	responseBodyBig
)

const TextPlain = "text/plain; charset=utf-8"

type Response struct {
	Status    StatusCode
	Headers   HeadersMap // keys are in lower case
	omitBody  bool       // don't add body and content-* headers for HEAD request
	bodyType  responseBodyType
	bodySmall []byte      // small body is sent in one write() call with headers
	bodyBig   io.WriterTo // big body, sent as multiple write() calls to socket
}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (conn *Connection) CreateResponse(
	status StatusCode, req *Request, wantsClose bool,
) *Response {
	res := &Response{
		Status: status,
		Headers: HeadersMap{
			"date": time.Now().UTC().Format(TimeFormat),
		},
	}

	res.omitBody = req.Method == "HEAD"
	if !res.omitBody &&
		status >= 200 &&
		status != StatusNoContent &&
		status != StatusNotModified {
		// content-length 0 is mandatory for most responses
		res.Headers["content-length"] = "0"
	}

	if wantsClose || !req.WantsKeepAlive {
		res.Headers["connection"] = "close"
	} else if req.Protocol == "HTTP/1.0" {
		res.Headers["connection"] = "keep-alive"
	}

	return res
}

func (res *Response) AttachBodySmall(body []byte, contentType string) {
	if res.omitBody {
		return
	}

	if res.bodyType != responseBodyEmpty {
		panic("response already has body")
	}
	res.bodyType = responseBodySmall
	res.bodySmall = body
	lenStr := strconv.FormatUint(uint64(len(body)), 10)
	res.Headers["content-length"] = lenStr
	res.Headers["content-type"] = contentType
}
func (res *Response) AttachBodySmallStr(body string, contentType string) {
	res.AttachBodySmall(sh.UnsafeGetStrBytes(body), contentType)
}

func (res *Response) AttachBodyBig(body io.WriterTo,
	size uint64, contentType string) {

	if res.omitBody {
		return
	}
	if res.bodyType != responseBodyEmpty {
		panic("response already has body")
	}
	res.bodyType = responseBodyBig
	res.bodyBig = body
	lenStr := strconv.FormatUint(size, 10)
	res.Headers["content-length"] = lenStr
	res.Headers["content-type"] = contentType
}

var ErrInvalidHeader = errors.New("http: header key or value contains CR or LF")

func (conn *Connection) SendResponse(res *Response) error {
	sb := &conn.wbuf
	sb.Reset()

	// first line
	// space before \r\n is required by standard
	sb.AddS("HTTP/1.1 ").AddU(uint64(res.Status)).AddS(" \r\n")

	// headers
	for k, v := range res.Headers {
		// prevent response splitting / header injection
		if strings.ContainsAny(k, "\r\n:") || strings.ContainsAny(v, "\r\n") {
			return ErrInvalidHeader
		}
		sb.AddS(k).AddS(": ").AddS(v).AddS("\r\n")
	}
	// empty line
	sb.AddS("\r\n")

	switch res.bodyType {
	case responseBodyEmpty:
		// net.Conn.Write follows the io.Writer contract: it returns a non-nil
		// error whenever n < len(p), so no loop is needed.
		if _, err := conn.socket.Write(conn.wbuf.Buf); err != nil {
			return err
		}
	case responseBodySmall:
		// write headers and body in one socket.Write() call
		bbb := net.Buffers{sb.Buf, res.bodySmall}
		if _, err := bbb.WriteTo(conn.socket); err != nil {
			return err
		}
	case responseBodyBig:
		// write headers, then write big body
		if _, err := conn.socket.Write(conn.wbuf.Buf); err != nil {
			return err
		}
		if _, err := res.bodyBig.WriteTo(conn.socket); err != nil {
			return err
		}
	}
	return nil
}

func (conn *Connection) SendErrorResponse(
	httpErr *HttpError, req *Request, wantsClose bool,
) error {

	res := conn.CreateResponse(httpErr.Status, req, wantsClose)
	// attach error message if method allows
	res.AttachBodySmallStr(httpErr.Error(), TextPlain)
	return conn.SendResponse(res)
}
