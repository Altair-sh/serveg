package http

import "errors"

// Usage:
//
//	err := conn.ReadRequest() / CreateResponse()
//	if httpErr, ok := errors.AsType[*http.HttpError](err); ok {
//		// Request is malformed, socket buffer contains unread garbage data.
//		// Send error code and close connection.
//	} else {
//		// something bad happened with the connection, just close it
//	}
type HttpError struct {
	Status StatusCode
	Msg    string
}

func (e *HttpError) Error() string {
	return e.Status.String() + ": " + e.Msg
}

func NewHttpError(code StatusCode, msg string) *HttpError {
	return &HttpError{Status: code, Msg: msg}
}

var ErrHasUnreadBody = NewHttpError(StatusRequestEntityTooLarge,
	"server didn't fully read request body, connection corrupted")

var ErrBodyHasBeenAlreadyRead = errors.New(
	"request body has already been read")

var ErrTransferEncodingNotSupported = NewHttpError(StatusBadRequest,
	"header transfer-encoding is not supported")

var ErrLineEndNotFound = NewHttpError(StatusBadRequest,
	"line end not found")

var ErrHeadersTooLarge = NewHttpError(StatusRequestHeaderFieldsTooLarge,
	"headers too large")

var ErrNotHttpRequest = NewHttpError(StatusBadRequest,
	"not HTTP request")

var ErrProtocolNotSupported = NewHttpError(StatusHTTPVersionNotSupported,
	"request protocol not supported")

var ErrUnsafeRequestPath = NewHttpError(StatusForbidden,
	"unsafe request path")

var ErrLineNotHeader = NewHttpError(StatusBadRequest,
	"not a header line")

var ErrHeaderNameStartsOrEndsWithSpace = NewHttpError(StatusBadRequest,
	"header name must not start or end with space")

var ErrHeaderKeyOrValueHasCRLF = errors.New("response header key or value contains CR or LF")
