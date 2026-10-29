package server

import (
	"net"
	"serveg/internal/log"
	"strconv"
	"time"
)

func Serve(conf *Config) error {
	logger := log.LoggerContext{Name: "server"}
	addr := net.JoinHostPort(conf.Ip.String(), strconv.Itoa(int(conf.Port)))

	logger.Info("listening at", addr)
	srv, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer func() { _ = srv.Close() }()

	for i := int64(0); ; i++ {
		conn, err := srv.Accept()
		if err != nil {
			logger.Warn(err)
			continue
		}
		go handleConnection(conn, i)
	}
}

func handleConnection(conn net.Conn, n int64) {
	logger := log.LoggerContext{Name: "conn-" + strconv.FormatInt(n, 10)}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic: ", r)
		}
		_ = conn.Close()
	}()

	buf := make([]byte, 4096)

	for {
		if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			logger.Warn(err)
			break
		}
		n, err := conn.Read(buf)
		if err != nil {
			logger.Warn(err)
			break
		}
		msgBytes := buf[:n]
		logger.Info("msg: `", string(msgBytes), "`")
		if _, err := conn.Write(msgBytes); err != nil {
			logger.Warn(err)
			break
		}
	}
}
