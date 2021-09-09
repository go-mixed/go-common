package server

import (
	"errors"
	"fmt"
	"go-common/utils"
	"net"
	"time"
)

type SimpleUDPClient struct {
	addr   string
	conn   net.Conn
	logger utils.ILogger
}

func NewSimpleUDPClient(addr string, logger utils.ILogger) (*SimpleUDPClient, error) {
	var err error
	client := &SimpleUDPClient{
		addr:   addr,
		logger: logger,
	}

	client.conn, err = net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *SimpleUDPClient) SimpleWrite(codec Codec, data []byte) error {
	s := SimpleData{Codec: codec, Data: data}

	buf, err := s.MarshalBinary()
	if err != nil {
		return err
	}
	if len(buf) > 1024 {
		return errors.New("the length of a simple-udp packet cannot > 1024")
	}

	_, err = c.Write(buf)
	return err
}

func (c *SimpleUDPClient) SimpleRead() (*SimpleData, error) {
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		return nil, err
	}

	s := SimpleData{}
	if err = s.UnmarshalBinary(buf[:n]); err != nil {
		return nil, fmt.Errorf("cannot unmarshal binary: [%x], error: %w", buf, err)
	}

	return &s, nil
}

func (c *SimpleUDPClient) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *SimpleUDPClient) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *SimpleUDPClient) Close() error {
	return c.conn.Close()
}

func (c *SimpleUDPClient) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *SimpleUDPClient) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *SimpleUDPClient) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *SimpleUDPClient) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *SimpleUDPClient) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
