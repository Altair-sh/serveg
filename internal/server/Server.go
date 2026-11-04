package server

import (
	"errors"
	"net"
	"serveg/internal/log"
	"sync"
)

type Server struct {
	Conf     *Config
	logger   log.LoggerContext
	listener net.Listener
	wg       sync.WaitGroup
}

func NewServer(conf *Config) *Server {
	return &Server{
		Conf:   conf,
		logger: log.LoggerContext{Name: "server"},
	}
}

func (srv *Server) StartGoroutine() (err error) {
	srv.logger.Info("listening at ", srv.Conf.Addr)
	srv.listener, err = net.Listen("tcp", srv.Conf.Addr)
	if err == nil {
		srv.wg.Go(srv.serve)
	}
	return
}

func (srv *Server) serve() {
	srv.logger.Info("server started")
	for i := int64(0); ; i++ {
		accepted, err := srv.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				srv.logger.Info("listener socket closed")
				return
			}
			srv.logger.Error(err)
			continue
		}
		conn := NewTcpConnection(srv, accepted.(*net.TCPConn), i)
		go conn.Start()
	}
}

func (srv *Server) Stop() {
	srv.logger.Info("stopping server")
	_ = srv.listener.Close()
	srv.wg.Wait()
}
