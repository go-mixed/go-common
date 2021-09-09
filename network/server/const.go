package server

import (
	"encoding/binary"
	"errors"
	"gopkg.in/restruct.v1"
	"net"
)

type Codec uint16

// ErrServerClosed is returned by the Server's Serve,
var ErrServerClosed = errors.New("server: server closed")

type SimpleData struct {
	Codec   Codec  `struct:"uint16"`
	DataLen uint16 `struct:"uint16,sizeof=Data"`
	Data    []byte
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
	SimpleWrite(addr net.Addr, codec Codec, buf []byte) (int, error)
}

type SimpleUDPHandle interface {
	Serve(conn SimplePacketConn, addr net.Addr, buf []byte)
}

type SimpleUDPHandleFunc func(conn SimplePacketConn, addr net.Addr, buf []byte)

func (f SimpleUDPHandleFunc) Serve(conn SimplePacketConn, addr net.Addr, buf []byte) {
	f(conn, addr, buf)
}
