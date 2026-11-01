package http

import (
	"bufio"
	"net"
	sh "serveg/internal/string_helper"
	"strings"
)

const maxHeaderBytes = 16 * 1024
const initialWriteBufCap = 512

type mapStrStr map[string]string
type HeadersMap map[string]string

type Connection struct {
	socket *net.TCPConn
	r      *bufio.Reader
	wbuf   sh.StringBuilder
}

func NewConnection(socket *net.TCPConn) *Connection {
	return &Connection{
		socket: socket,
		r:      bufio.NewReaderSize(socket, maxHeaderBytes),
		wbuf:   sh.StringBuilder{Buf: make([]byte, 0, initialWriteBufCap)},
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
