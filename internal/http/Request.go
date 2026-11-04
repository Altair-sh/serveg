package http

import (
	"fmt"
	"io"
	"net/url"
	sh "serveg/internal/string_helper"
	"strconv"
	"strings"
)

type Request struct {
	needsReset bool
	Method     string
	Protocol   string
	// url-encoded characters (%D0), query params (?v=123)
	UrlRaw      string
	Url         string
	QueryParams mapStrStr
	// keys are in lower case
	// duplicate keys are joined with ", "
	Headers HeadersMap
	// if true, don't close connection after response
	WantsKeepAlive bool
	// value from header or 0
	ContentLength int64
	// wrapper of conn.r
	bodyReader io.LimitedReader
}

func (req *Request) reset() {
	if !req.needsReset {
		return
	}

	req.needsReset = false
	req.Method = req.Method[:0]
	req.Protocol = req.Protocol[:0]
	req.UrlRaw = req.UrlRaw[:0]
	req.Url = req.Url[:0]
	clear(req.QueryParams)
	clear(req.Headers)
	req.WantsKeepAlive = false
	req.ContentLength = 0
	req.bodyReader.N = 0
}

// Returns valid request pointer. If err != nil, req is partially initialized.
// Don't forget to read body with GetBodyReader() if HasUnreadBody() or ReadRequest will return error.
// See [HttpError]
func (conn *Connection) ReadRequest() (req *Request, err error) {
	req = &conn.req
	// TODO: make this check run automatically between ReadRequest and SendResponse.
	// This can be done by moving reading loop from server/connection to http/Connection and handling requestr in callback (req) => res
	if req.HasUnreadBody() {
		err = ErrHasUnreadBody
		return
	}
	req.reset()
	req.needsReset = true
	// TODO: remove after moving to callback loop
	conn.res.reset()

	var readN int
	// METHOD /path HTTP/ver
	if err = conn.readFirstLine(req, &readN); err != nil {
		return
	}
	// key: value
	// ...
	// empty line
	if err = conn.readHeaders(req, &readN); err != nil {
		return
	}

	// check body size and initialize req.Body
	transferEncoding := req.Headers["transfer-encoding"]
	if transferEncoding != "" {
		err = ErrTransferEncodingNotSupported
		return
	}
	contentLengthStr := req.Headers["content-length"]
	if contentLengthStr != "" {
		var contentLength uint64
		contentLength, err = strconv.ParseUint(contentLengthStr, 10, 64)
		if err != nil {
			err = NewHttpError(StatusBadRequest,
				"can't parse header: content-length: "+contentLengthStr)
			return
		}
		if contentLength > 0 {
			req.ContentLength = int64(contentLength)
			req.bodyReader.N = req.ContentLength
		}
	}

	// set WantsKeepAlive
	connectionStrSplit := strings.Split(req.Headers["connection"], ",")
	if req.Protocol == "HTTP/1.0" {
		req.WantsKeepAlive = HeaderValueContains(connectionStrSplit, "keep-alive")
	} else {
		req.WantsKeepAlive = !HeaderValueContains(connectionStrSplit, "close")
	}

	return
}

// if true CreateResponse will return error
func (req *Request) HasUnreadBody() bool {
	return req.bodyReader.N > 0
}

func (conn *Connection) GetBodyReader(maxBodyLen int64) (io.Reader, error) {
	req := &conn.req
	if req.ContentLength > maxBodyLen {
		return nil, NewHttpError(StatusRequestEntityTooLarge,
			fmt.Sprintf("content-length %d > maxBodyLen %d",
				req.ContentLength, maxBodyLen))
	}
	if !req.HasUnreadBody() {
		return nil, ErrBodyHasBeenAlreadyRead
	}

	if req.ContentLength != 0 {
		// body reader with N=0 is a valid reader that just returns EOF
		req.bodyReader.N = min(req.bodyReader.N, maxBodyLen)
	}
	return &req.bodyReader, nil
}

func (conn *Connection) readLineBelowLimit(readN *int) (string, error) {
	// read up to maxHeaderBytes
	slice, notLineEnd, err := conn.r.ReadLine()
	if err != nil {
		// something is wrong with tcp socket
		return "", err
	}
	if notLineEnd {
		// buffer end was reached but no '\n' found
		return "", ErrLineEndNotFound
	}

	*readN += len(slice) + 1 // \n
	if *readN > maxHeaderBytes {
		return "", ErrHeadersTooLarge
	}

	return strings.TrimRight(string(slice), " \r\n"), nil
}

func (conn *Connection) readFirstLine(req *Request, readN *int) error {
	line, err := conn.readLineBelowLimit(readN)
	if err != nil {
		return err
	}
	// skip one empty line
	// some clients send it between requests
	if line == "" {
		line, err = conn.readLineBelowLimit(readN)
		if err != nil {
			return err
		}
	}

	parts := strings.Split(line, " ")
	if len(parts) != 3 || !strings.HasPrefix(parts[2], "HTTP/") {
		return ErrNotHttpRequest
	}
	req.Method = parts[0]
	// url-encoded characters (%D0), query params (?v=123)
	req.UrlRaw = parts[1]
	req.Protocol = parts[2]

	if req.Protocol != "HTTP/1.0" && req.Protocol != "HTTP/1.1" {
		return ErrProtocolNotSupported
	}

	u, err := url.ParseRequestURI(req.UrlRaw)
	if err != nil {
		return NewHttpError(StatusBadRequest, err.Error())
	}

	req.Url = sh.ResolveSafePath(u.Path)
	if req.Url == "" {
		return ErrUnsafeRequestPath
	}
	for key, values := range u.Query() {
		req.QueryParams[key] = values[0]
	}

	return nil
}

func (conn *Connection) readHeaders(req *Request, readN *int) error {
	for {
		line, err := conn.readLineBelowLimit(readN)
		if err != nil {
			return err
		}
		// empty line means end of headers
		if line == "" {
			break
		}

		i := strings.IndexByte(line, ':')
		if i < 1 {
			return ErrLineNotHeader
		}
		key := line[:i]
		firstB, lastB := key[0], key[i-1]
		if firstB == ' ' || firstB == '\t' ||
			lastB == ' ' || lastB == '\t' {
			return ErrHeaderNameStartsOrEndsWithSpace
		}
		key = strings.ToLower(key)

		// val can be empty
		val := strings.TrimSpace(line[i+1:])
		// if header repeats, join values with ", "
		existing := req.Headers[key]
		if existing != "" {
			val = existing + ", " + val
		}
		req.Headers[key] = val
	}
	return nil
}
