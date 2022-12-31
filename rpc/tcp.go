package rpc

func NewTcpServer(host string, logger logger.ILogger) *Server {
	return NewServer("tcp", host, logger)
}

func NewTcpClient(host string, logger logger.ILogger) (*Client, error) {
	return NewClient("tcp", host, logger)
}
