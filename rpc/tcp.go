package rpc

import (
	"go-common/utils"
)

func NewTcpServer(host string, logger utils.ILogger) *Server {
	return NewServer("tcp", host, logger)
}

func NewTcpClient(host string, logger utils.ILogger) (*Client, error) {
	return NewClient("tcp", host, logger)
}
