package http

import (
	"bufio"
	"net"
	sh "serveg/internal/string_helper"
	"strings"
	"time"
)

const maxHeaderBytes = 16 * 1024
const initialWriteBufCap = 512

type mapStrStr map[string]string
type HeadersMap map[string]string

type Connection struct {
	socket  *net.TCPConn
	timeout time.Duration
	r       *bufio.Reader
	wbuf    sh.StringBuilder
	// hidden to prevent using before ReadRequest/SendResponse
	req Request  // reset by ReadRequest
	res Response // reset by CreateResponse and ReadRequest
}

func NewConnection(socket *net.TCPConn, timeout time.Duration) *Connection {
	return &Connection{
		socket:  socket,
		timeout: timeout,
		r:       bufio.NewReaderSize(socket, maxHeaderBytes),
		wbuf:    sh.StringBuilder{Buf: make([]byte, 0, initialWriteBufCap)},
		req: Request{
			QueryParams: make(mapStrStr),
			Headers:     make(HeadersMap),
		},
		res: Response{
			Headers: make(HeadersMap),
		},
	}
}

func HeaderValueContains(sourceSplit []string, target string) bool {
	for _, v := range sourceSplit {
		v = strings.TrimSpace(v)
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}
