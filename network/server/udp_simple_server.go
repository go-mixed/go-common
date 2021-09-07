package server

import (
	"encoding/binary"
	"go-common/utils"
	"net"
)

type SimpleUDPServer struct {
	host   string
	logger utils.ILogger
	codecs map[Codec]SimpleUDPHandle
}

type SimpleUDPHandle interface {
	Serve(conn net.PacketConn, addr net.Addr, buf []byte)
}

type SimpleUDPHandleFunc func(conn net.PacketConn, addr net.Addr, buf []byte)

func (f SimpleUDPHandleFunc) Serve(conn net.PacketConn, addr net.Addr, buf []byte) {
	f(conn, addr, buf)
}

// NewSimpleUDPServer 一个简单的udp server， 注意：只能针对单个包来解析，并不会合并包
// 格式为 | 2 bytes codec | data | 一个包的长度不要超过1024
func NewSimpleUDPServer(host string, logger utils.ILogger) *SimpleUDPServer {
	return &SimpleUDPServer{
		host:   host,
		logger: logger,
		codecs: map[Codec]SimpleUDPHandle{},
	}
}

func (s *SimpleUDPServer) Run(stopChan <-chan bool) error {
	s.logger.Infof("run udp server on %s", s.host)

	packetConn, err := net.ListenPacket("udp", s.host)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-stopChan:
			packetConn.Close()
			s.logger.Infof("close udp server on %s", s.host)
		}
	}()

	for {
		// UDP包最大长度: 65535- IP头(20) - UDP头(8)＝65507字节
		// 但是因MTU=1500的限制(拨号连接的MTU是1280)，一个包最多1472长度，超过就要分割
		// 考虑到业务形态, 这里规定一个包长度最大1024
		buf := make([]byte, 1024)
		n, remoteAddr, err := packetConn.ReadFrom(buf)
		if err != nil {
			select {
			case <-stopChan: // 外部关闭了服务器
				return ErrServerClosed
			default:
				return err
			}
		}
		go s.serve(packetConn, remoteAddr, buf[:n])
	}
}

func (s *SimpleUDPServer) RegisterCodec(codec Codec, callback SimpleUDPHandle) {
	s.codecs[codec] = callback
}

func (s *SimpleUDPServer) serve(conn net.PacketConn, remoteAddr net.Addr, buf []byte) {
	if len(buf) < 2 {
		s.logger.Warnf("udp package length must >= 2")
		return
	}
	codec := binary.BigEndian.Uint16(buf[:2])
	if callback, ok := s.codecs[Codec(codec)]; ok {
		callback.Serve(conn, remoteAddr, buf[2:])
	} else {
		s.logger.Warnf("unknown codec, invalid udp package: %x", buf)
	}
}
