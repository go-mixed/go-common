package http_utils

import (
	"context"
	"go-common/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type HttpServer struct {
	serveMux *http.ServeMux
	server   *http.Server
	logger   utils.ILogger
}

func NewHttpServer(address string, logger utils.ILogger) *HttpServer {
	serveMux := http.NewServeMux()
	return &HttpServer{
		serveMux: serveMux,
		server: &http.Server{
			Addr:    address,
			Handler: serveMux,
		},
		logger: logger,
	}
}

func (s *HttpServer) Handle(pattern string, handler http.Handler) {
	s.serveMux.Handle(pattern, handler)
}

func (s *HttpServer) SetServeMux(serveMux *http.ServeMux) {
	s.serveMux = serveMux
	s.server.Handler = s.serveMux
}

func (s *HttpServer) SetDefaultServeMux() {
	s.serveMux = http.DefaultServeMux
}

func (s *HttpServer) GetServeMux() *http.ServeMux {
	return s.serveMux
}

func (s *HttpServer) SetNativeServer(server *http.Server) {
	s.server = server
	if s.server.Handler == nil {
		s.server.Handler = s.serveMux
	}
}

func (s *HttpServer) GetNativeServer() *http.Server {
	return s.server
}

// 监听停止信号, ctx为nil时只收听进程退出信号
func (s *HttpServer) listenContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx1, cancel := context.WithCancel(ctx)
		termChan := make(chan os.Signal)
		//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
		signal.Notify(termChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			select {
			case <-termChan:
				s.logger.Info("exit signal of process received.")
				cancel()
			}
		}()
		return ctx1
	} else {
		return ctx
	}
}

func (s *HttpServer) Run(ctx context.Context) error {
	go func() {
		select {
		case <-s.listenContext(ctx).Done():
		}

		if err := s.server.Close(); err != nil {
			s.logger.Fatalf("Server closed: %s", err.Error())
		}
	}()

	if err := s.server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			s.logger.Info("http server closed")
		} else {
			return err
		}
	}

	return nil
}
