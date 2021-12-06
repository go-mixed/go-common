package rpc

import (
	"go-common/utils"
	"go-common/utils/core"
	"go-common/utils/io"
	"go-common/utils/text"
	"os"
	"path/filepath"
	"strings"
)

// NewUdsServer Unix domain socket supported
//- Unix, Unix-like, Linux
//- Windows 10 Insider Build 17063+(Windows 10 version 1809 (aka the October 2018 Update))
//- Windows Server 1809/2019+
func NewUdsServer(unixSockFile string, logger utils.ILogger) *Server {
	unixSockFile = cleanSockFile(unixSockFile)
	_ = os.Remove(unixSockFile)
	return NewServer("unix", unixSockFile, logger)
}

func NewUdsClient(unixSockFile string, logger utils.ILogger) (*Client, error) {
	unixSockFile = cleanSockFile(unixSockFile)
	return NewClient("unix", unixSockFile, logger)
}

func cleanSockFile(unixSockFile string) string {
	// 注意: 在WSL中, 如果sock文件放在/mnt/下，UDS将会无法运行
	// 保存linux系统的其它目录则不会有问题：https://github.com/Microsoft/WSL/issues/2137

	if core.IsInWSL() {
		abs, err := filepath.Abs(unixSockFile)
		if err != nil {
			return unixSockFile
		}
		if strings.HasPrefix(abs, "/mnt/") { // 重定向/tmp目录
			unixSockFile = filepath.Join(os.TempDir(), text_utils.Md5(abs)+".sock")
		}
	}

	return unixSockFile
}

func GetUdsFile(file string) string {
	return filepath.Join(io_utils.GetCurrentDir(), "run", file+".sock")
}
