package rpc

import (
	"go-common/utils"
	"os"
)

// Unix domain socket supported
//- Unix, Unix-like, Linux
//- Windows 10 Insider Build 17063+(Windows 10 version 1809 (aka the October 2018 Update))
//- Windows Server 1809/2019+

func NewUdsServer(unixSockFile string, logger utils.ILogger) *Server {
	_ = os.Remove(unixSockFile)
	return NewServer("unix", unixSockFile, logger)
}

func NewUdsClient(unixSocketFile string, logger utils.ILogger) (*Client, error) {
	return NewClient("unix", unixSocketFile, logger)
}
