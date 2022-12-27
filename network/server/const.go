package server

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"gopkg.in/restruct.v1"
	"net"
)

const (
	MaxLanUdpLen      = 1472 // 1500-8-20
	MaxInternetUdpLen = 548  // (576-8-20)
)

type Codec uint16

// ErrServerClosed is returned by the Server's Serve,
var ErrServerClosed = errors.New("server: server closed")

type SimpleData struct {
	Codec     Codec  `struct:"uint16"`
	MessageID uint32 `struct:"uint32"` // 溢出了从0开始
	DataLen   uint16 `struct:"uint16,sizeof=Data"`
	Data      []byte
}

func (s *SimpleData) MarshalBinary() ([]byte, error) {
	return restruct.Pack(binary.BigEndian, s)
}

func (s *SimpleData) UnmarshalBinary(data []byte) error {
	return restruct.Unpack(data, binary.BigEndian, s)
}

type SimplePacketConn interface {
	net.PacketConn

	SimpleRead() (*SimpleData, net.Addr, error)
	SimpleWrite(addr net.Addr, codec Codec, messageID uint32, buf []byte) (int, error)
}

type SimpleUDPHandle interface {
	Serve(conn SimplePacketConn, addr net.Addr, data *SimpleData)
}

type SimpleUDPHandleFunc func(conn SimplePacketConn, addr net.Addr, data *SimpleData)

func (f SimpleUDPHandleFunc) Serve(conn SimplePacketConn, addr net.Addr, data *SimpleData) {
	f(conn, addr, data)
}
