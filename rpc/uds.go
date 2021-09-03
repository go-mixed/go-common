package rpc

import (
	"go-common/utils"
	"os"
)

func NewUdsServer(unixSockFile string, logger utils.ILogger) *Server {
	_ = os.Remove(unixSockFile)
	return NewServer("unix", unixSockFile, logger)
}

func NewUdsClient(unixSocketFile string, logger utils.ILogger) (*Client, error) {
	return NewClient("unix", unixSocketFile, logger)
}
