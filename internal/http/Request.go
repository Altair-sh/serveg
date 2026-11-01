package http

import (
	"net/url"
	sh "serveg/internal/string_helper"
	"strings"
)

type Request struct {
	Method   string
	Protocol string
	// url-encoded characters (%D0), query params (?v=123)
	PathRaw     string
	Path        string
	QueryParams mapStrStr
	// keys are in lower case
	// duplicate keys are joined with ", "
	Headers HeadersMap
	// if true, don't close connection after response
	WantsKeepAlive bool
}

// Read request (body not supported)
// Returns valid request pointer. If err != nil, req is partially parsed.
// It doesn't enforce "Host" header because i don't need it in this project.
//
//	if httpErr, ok := errors.AsType[*http.HttpError](err); ok {
//		// Request is malformed, socket buffer contains unread garbage data.
//		// Send error code and close connection.
//	} else {
//		// something bad happened with the connection, just close it
//	}
func (conn *Connection) ReadRequest() (req *Request, err error) {
	req = new(Request)
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

	// This server doesn't accept request bodies.
	// Reject connection rather than desync the stream.
	contentLength := req.Headers["content-length"]
	transferEncoding := req.Headers["transfer-encoding"]
	if (contentLength != "" && contentLength != "0") ||
		transferEncoding != "" {
		err = NewHttpError(StatusBadRequest,
			"request bodies are not supported")
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

func (conn *Connection) readLineBelowLimit(readN *int) (string, error) {
	// read up to maxHeaderBytes
	slice, notLineEnd, err := conn.r.ReadLine()
	if err != nil {
		// something is wrong with tcp socket
		return "", err
	}
	if notLineEnd {
		// buffer end was reached but no '\n' found
		return "", NewHttpError(StatusBadRequest, "line end not found")
	}

	*readN += len(slice) + 1 // \n
	if *readN > maxHeaderBytes {
		return "", NewHttpError(StatusRequestHeaderFieldsTooLarge,
			"headers too large")
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
		return NewHttpError(StatusBadRequest,
			"malformed request line '"+line+"'")
	}
	req.Method = parts[0]
	// url-encoded characters (%D0), query params (?v=123)
	req.PathRaw = parts[1]
	req.Protocol = parts[2]

	if req.Protocol != "HTTP/1.0" && req.Protocol != "HTTP/1.1" {
		return NewHttpError(StatusHTTPVersionNotSupported,
			"request protocol not supported: "+req.Protocol)
	}

	u, err := url.ParseRequestURI(req.PathRaw)
	if err != nil {
		return NewHttpError(StatusBadRequest, err.Error())
	}

	req.Path = sh.ResolveSafePath(u.Path)
	if req.Path == "" {
		return NewHttpError(StatusForbidden,
			"unsafe request path '"+u.Path+"'")
	}
	req.QueryParams = make(mapStrStr)
	for key, values := range u.Query() {
		req.QueryParams[key] = values[0]
	}

	return nil
}

func (conn *Connection) readHeaders(req *Request, readN *int) error {
	req.Headers = make(HeadersMap)
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
			return NewHttpError(StatusBadRequest,
				"not a header line '"+line+"'")
		}
		key := line[:i]
		firstB, lastB := key[0], key[i-1]
		if firstB == ' ' || firstB == '\t' ||
			lastB == ' ' || lastB == '\t' {
			return NewHttpError(StatusBadRequest,
				"header name must not start or end with space")
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
