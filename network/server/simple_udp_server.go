package server

import (
	"encoding/binary"
	"go-common/utils"
	"net"
	"time"
)

type SimpleUDPServer struct {
	host       string
	logger     utils.ILogger
	codecs     map[Codec]SimpleUDPHandle
	packetConn net.PacketConn
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

	var err error
	s.packetConn, err = net.ListenPacket("udp", s.host)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-stopChan:
			s.packetConn.Close()
			s.logger.Infof("close udp server on %s", s.host)
		}
	}()

	for {
		// UDP包最大长度: 65535- IP头(20) - UDP头(8)＝65507字节
		// 但是因MTU=1500的限制(拨号连接的MTU是1280)，一个包最多1472长度，超过就要分割
		// 考虑到业务形态, 这里规定一个包长度最大1024
		buf := make([]byte, 1024)
		n, remoteAddr, err := s.ReadFrom(buf)
		if err != nil {
			select {
			case <-stopChan: // 外部关闭了服务器
				return ErrServerClosed
			default:
				return err
			}
		}
		go s.serve(s.packetConn, remoteAddr, buf[:n])
	}
}

func (s *SimpleUDPServer) RegisterCodec(codec Codec, callback SimpleUDPHandle) {
	s.codecs[codec] = callback
}

func (s *SimpleUDPServer) serve(conn net.PacketConn, remoteAddr net.Addr, buf []byte) {
	if len(buf) < 2 {
		s.logger.Warnf("the length of simple-udp package must >= 2")
		return
	}

	codec := binary.BigEndian.Uint16(buf[:2])
	if callback, ok := s.codecs[Codec(codec)]; ok {
		_s := SimpleData{}
		if err := _s.UnmarshalBinary(buf); err != nil {
			s.logger.Error("invalid package format: %s, raw: %x", err.Error(), buf)
			return
		}
		callback.Serve(s, remoteAddr, _s.Data)
	} else {
		s.logger.Warnf("unknown codec, raw: %x", buf)
	}
}

func (s *SimpleUDPServer) SimpleRead() (*SimpleData, net.Addr, error) {
	buf := make([]byte, 1024)
	n, addr, err := s.ReadFrom(buf)
	if err != nil {
		return nil, nil, err
	}

	_s := SimpleData{}
	if err = _s.UnmarshalBinary(buf[:n]); err != nil {
		return nil, nil, err
	}

	return &_s, addr, nil
}

func (s *SimpleUDPServer) SimpleWrite(addr net.Addr, codec Codec, buf []byte) (int, error) {
	_s := SimpleData{Codec: codec, Data: buf}
	b, err := _s.MarshalBinary()
	if err != nil {
		return 0, err
	}
	return s.WriteTo(b, addr)
}

func (s *SimpleUDPServer) ReadFrom(p []byte) (int, net.Addr, error) {
	return s.packetConn.ReadFrom(p)
}

func (s *SimpleUDPServer) WriteTo(p []byte, addr net.Addr) (int, error) {
	return s.packetConn.WriteTo(p, addr)
}

func (s *SimpleUDPServer) LocalAddr() net.Addr {
	return s.packetConn.LocalAddr()
}

func (s *SimpleUDPServer) SetDeadline(t time.Time) error {
	return s.packetConn.SetDeadline(t)
}

func (s *SimpleUDPServer) SetReadDeadline(t time.Time) error {
	return s.packetConn.SetReadDeadline(t)
}

func (s *SimpleUDPServer) SetWriteDeadline(t time.Time) error {
	return s.packetConn.SetWriteDeadline(t)
}

func (s *SimpleUDPServer) Close() error {
	if s.packetConn != nil {
		return s.packetConn.Close()
	}
	return nil
}
