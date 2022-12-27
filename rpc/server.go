package rpc

import (
	"context"
	"github.com/pkg/errors"
	"go-common/utils"
	"go-common/utils/core"
	"go-common/utils/http"
	"net"
	"net/http"
	"net/rpc"
)

type Server struct {
	network string
	address string
	logger  utils.ILogger
	*rpc.Server
}

// ErrServerClosed is returned by the Server's Serve, ServeTLS, ListenAndServe,
// and ListenAndServeTLS methods after a call to Shut down or Close.
var ErrServerClosed = errors.New("rpc: Server closed")

func NewServer(network, address string, logger utils.ILogger) *Server {
	return &Server{
		network,
		address,
		logger,
		rpc.NewServer(),
	}
}

func NewHttpServer(address string, logger utils.ILogger) *Server {
	return NewServer("http", address, logger)
}

func (s *Server) Registers(methods ...any) error {
	for _, method := range methods {
		if err := rpc.Register(method); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	if s.network == "http" {
		return s.runHttp(ctx)
	} else {
		return s.runNormal(ctx)
	}
}

func (s *Server) runNormal(ctx context.Context) error {
	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		return err
	}

	s.logger.Infof("start rpc-server(via tcp) on [%s]%s", s.network, s.address)

	// 监听并关闭监听
	go func() {
		core.WaitForStopped(ctx.Done())
		listener.Close()
		s.logger.Infof("stop rpc-server(via tcp) on [%s]%s", s.network, s.address)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if core.IsContextDone(ctx) {
				return ErrServerClosed
			}

			s.logger.Errorf("rpc.Serve(via tcp): accept:", err.Error())
			break
		}
		go s.ServeConn(conn)
	}

	s.logger.Infof("rpc server(via tcp) quit on [%s]%s", s.network, s.address)

	return nil
}

func (s *Server) runHttp(ctx context.Context) error {

	server := http_utils.NewHttpServer(s.address, s.logger)

	s.logger.Infof("start rpc-server(via http) on [%s]%s", s.network, s.address)

	oldDefaultServeMux := http.DefaultServeMux             // 存储旧的全局DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()              // 新建一个空白serveMux到全局的DefaultServeMux
	s.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath) // 设置路由到新的defaultServeMux (http.Handle就是设置到DefaultServeMux)
	server.SetServeMux(http.DefaultServeMux)               // 设置当前server为这个新的DefaultServeMux
	http.DefaultServeMux = oldDefaultServeMux              // 恢复全局的DefaultServeMux

	// 因为无法得到debugHTTP, 所以只能使用上面的方法 并且还无法线程安全
	//server.Handle(rpc.DefaultRPCPath, s)
	//server.Handle(rpc.DefaultDebugPath, rpc.debugHTTP{s})

	if err := server.Run(ctx); err != nil {
		return err
	}

	s.logger.Infof("stop rpc-server(via http) on [%s]%s", s.network, s.address)

	return nil
}
