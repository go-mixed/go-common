package rpc

import (
	"errors"
	"go-common/utils"
	"go-common/utils/core"
	"net"
	"net/rpc"
)

type Server struct {
	network string
	address string
	logger  utils.ILogger
	*rpc.Server
}

// ErrServerClosed is returned by the Server's Serve, ServeTLS, ListenAndServe,
// and ListenAndServeTLS methods after a call to Shutdown or Close.
var ErrServerClosed = errors.New("rpc: Server closed")

func NewServer(network, address string, logger utils.ILogger) *Server {
	return &Server{
		network,
		address,
		logger,
		rpc.NewServer(),
	}
}

func (s *Server) Registers(methods ...interface{}) error {
	for _, method := range methods {
		if err := rpc.Register(method); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Run(stopChan <-chan bool) error {
	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}

	s.logger.Infof("start rpc-server on [%s]%s", s.network, s.address)

	// 监听并关闭监听
	go func() {
		core.WaitForStopped(stopChan)
		listener.Close()
		s.logger.Infof("stop rpc-server on [%s]%s", s.network, s.address)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if core.IsStopped(stopChan) {
				return ErrServerClosed
			}

			s.logger.Errorf("rpc.Serve: accept:", err.Error())
			break
		}
		go s.ServeConn(conn)
	}

	s.logger.Infof("rpc server quit on [%s]%s", s.network, s.address)

	return nil
}
