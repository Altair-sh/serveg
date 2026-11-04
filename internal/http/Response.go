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
	needsReset bool
	Status     StatusCode
	// keys are in lower case
	Headers HeadersMap
	// don't add body and content-* headers for HEAD request
	omitBody bool
	bodySize int64
	bodyType responseBodyType
	// small body is sent in one write() call with headers
	bodyBuf []byte
	// big body, sent as multiple write() calls to socket
	bodyReader io.Reader
}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (res *Response) reset() {
	if !res.needsReset {
		return
	}

	res.needsReset = false
	res.Status = StatusNotInitialized
	clear(res.Headers)
	res.omitBody = false
	res.bodySize = 0
	res.bodyType = responseBodyEmpty
	res.bodyBuf = res.bodyBuf[:0]
	if closer, ok := res.bodyReader.(io.Closer); ok {
		// body is owned by the response
		_ = closer.Close()
	}
	res.bodyReader = nil
}

func (conn *Connection) CreateResponse(status StatusCode, req *Request, wantsClose bool) *Response {

	res := &conn.res
	res.reset()
	res.needsReset = true

	res.Status = status
	res.Headers["date"] = time.Now().UTC().Format(TimeFormat)
	// without it browser may remember response and not resend it
	res.Headers["cache-control"] = "no-store"

	res.omitBody = req.Method == "HEAD" ||
		status < 200 ||
		status == StatusNoContent ||
		status == StatusResetContent ||
		status == StatusNotModified
	if !res.omitBody {
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

func (res *Response) WithBodyBuf(body []byte, contentType string) *Response {
	if res.omitBody {
		return res
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
	return res
}
func (res *Response) WithBodyStr(body string, contentType string) *Response {
	return res.WithBodyBuf(sh.UnsafeGetStrBytes(body), contentType)
}

// body will be closed by SendResponse() or reset()
func (res *Response) WithBodyReader(body io.Reader,
	size int64, contentType string,
) *Response {
	if res.omitBody {
		return res
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
	return res
}

func (conn *Connection) SendResponse(res *Response) (err error) {
	if closer, ok := res.bodyReader.(io.Closer); ok {
		// body is owned by the response
		defer func() {
			_ = closer.Close()
		}()
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
			return ErrHeaderKeyOrValueHasCRLF
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
		// WithBodyReader was called.
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
	return conn.CreateResponse(httpErr.Status, req, wantsClose).
		WithBodyStr(httpErr.Error(), TextPlain)
}
