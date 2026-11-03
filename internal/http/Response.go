package http

import (
	"errors"
	"fmt"
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
	responseBodyBuf
	responseBodyReader
)

type Response struct {
	Status     StatusCode
	Headers    HeadersMap // keys are in lower case
	omitBody   bool       // don't add body and content-* headers for HEAD request
	bodySize   int64
	bodyType   responseBodyType
	bodyBuf    []byte    // small body is sent in one write() call with headers
	bodyReader io.Reader // big body, sent as multiple write() calls to socket
}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (conn *Connection) CreateResponse(
	status StatusCode, req *Request, wantsClose bool,
) *Response {
	res := &Response{
		Status: status,
		Headers: HeadersMap{
			"date": time.Now().UTC().Format(TimeFormat),
			// without it browser may remember response and not resend it
			"cache-control": "no-store",
		},
	}

	res.omitBody = req.Method == "HEAD"
	if !res.omitBody &&
		status >= 200 &&
		status != StatusNoContent &&
		status != StatusNotModified {
		// content-length is mandatory for most responses
		res.Headers["content-length"] = "0"
	}

	if wantsClose || !req.WantsKeepAlive {
		res.Headers["connection"] = "close"
	} else if req.Protocol == "HTTP/1.0" {
		res.Headers["connection"] = "keep-alive"
	}

	return res
}

func (res *Response) AttachBodyBuf(body []byte, contentType string) {
	if res.omitBody {
		return
	}

	if res.bodyType != responseBodyEmpty {
		panic("response already has body")
	}
	res.bodyType = responseBodyBuf
	res.bodyBuf = body
	size := int64(len(body))
	res.bodySize = size
	lenStr := strconv.FormatUint(uint64(size), 10)
	res.Headers["content-length"] = lenStr
	res.Headers["content-type"] = contentType
}
func (res *Response) AttachBodyStr(body string, contentType string) {
	res.AttachBodyBuf(sh.UnsafeGetStrBytes(body), contentType)
}

// body will be closed by SendResponse
func (res *Response) AttachBodyReader(body io.Reader,
	size int64, contentType string) {

	if res.omitBody {
		return
	}
	if res.bodyType != responseBodyEmpty {
		panic("response already has body")
	}
	res.bodyType = responseBodyReader
	res.bodyReader = body
	res.bodySize = size
	lenStr := strconv.FormatUint(uint64(size), 10)
	res.Headers["content-length"] = lenStr
	res.Headers["content-type"] = contentType
}

var ErrInvalidHeader = errors.New("http: header key or value contains CR or LF")

func (conn *Connection) SendResponse(res *Response) (err error) {
	if res.bodyType == responseBodyReader {
		if closer, ok := res.bodyReader.(io.Closer); ok {
			// body is owned by the response
			defer func() {
				_ = closer.Close()
			}()
		}
	}

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
		_, err = conn.socket.Write(conn.wbuf.Buf)
	case responseBodyBuf:
		// write headers and body in one socket.Write() call
		bufs := net.Buffers{sb.Buf, res.bodyBuf}
		_, err = bufs.WriteTo(conn.socket)
	case responseBodyReader:
		// write headers
		if _, err = conn.socket.Write(conn.wbuf.Buf); err != nil {
			return
		}
		// Write exactly bodySize bytes from body.
		// It prevents session corruption if body size changed since
		// AttachBodyReader was called.
		remaining := res.bodySize
		for remaining > 0 {
			err = conn.socket.SetWriteDeadline(time.Now().Add(conn.timeout))
			if err != nil {
				return
			}
			// don't increase or slow connections will trigger socket deadline
			// 256 KB / 30 s ≈ 8.7 KB/s
			const fileReadChunkSize = 256 * 1024
			n := min(remaining, fileReadChunkSize)
			_, err = io.CopyN(conn.socket, res.bodyReader, n)
			if err != nil {
				if errors.Is(err, io.EOF) {
					// Body ended before bodySize bytes.
					// Returning just io.EOF would look like a client discnnect
					err = fmt.Errorf("response body is %d bytes shorter"+
						" than content-length: %w",
						remaining, io.ErrUnexpectedEOF)
				}
				return
			}
			remaining -= n
		}
	}
	return
}

func (conn *Connection) CreateErrorResponse(
	req *Request, wantsClose bool, httpErr *HttpError,
) *Response {

	res := conn.CreateResponse(httpErr.Status, req, wantsClose)
	// attach error message if method allows
	res.AttachBodyStr(httpErr.Error(), TextPlain)
	return res
}
