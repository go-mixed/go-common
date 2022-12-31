package rpc

import "gopkg.in/go-mixed/go-common.v1/utils"

func NewTcpServer(host string, logger utils.ILogger) *Server {
	return NewServer("tcp", host, logger)
}

func NewTcpClient(host string, logger utils.ILogger) (*Client, error) {
	return NewClient("tcp", host, logger)
}
