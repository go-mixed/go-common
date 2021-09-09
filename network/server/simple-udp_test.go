package server

import (
	"go-common/utils"
	"net"
	"testing"
	"time"
)

func TestSimpleUDP(t *testing.T) {
	server := NewSimpleUDPServer("0.0.0.0:99", utils.NewDefaultLogger())
	a := SimpleUDPHandleFunc(func(conn SimplePacketConn, addr net.Addr, buf []byte) {
		time.Sleep(1 * time.Second)
		conn.SimpleWrite(addr, 0xc2c2, append([]byte("pong: "), buf...))
	})
	server.RegisterCodec(0xc1c2, a)

	stopChan := make(chan bool)
	go func() {
		server.Run(stopChan)
	}()
	time.Sleep(time.Second)
	defer func() {
		stopChan <- true
	}()

	client, err := NewSimpleUDPClient("127.0.0.1:99", utils.NewDefaultLogger())
	if err != nil {
		t.Errorf("client connect error: %s", err.Error())
	}

	if err = client.SimpleWrite(0xc1c2, []byte("ping1")); err != nil {
		t.Errorf("client send error: %s", err.Error())
	}

	if err = client.SimpleWrite(0xc1c2, []byte("ping2")); err != nil {
		t.Errorf("client send error: %s", err.Error())
	}
	client.SetReadDeadline(time.Now().Add(1500 * time.Millisecond))

	if s, err := client.SimpleRead(); err != nil {
		t.Errorf("client read %s", err.Error())
	} else {
		t.Logf("recv %x, %s", s.Codec, s.Data)
	}

	if s, err := client.SimpleRead(); err != nil {
		t.Errorf("client read %s", err.Error())
	} else {
		t.Logf("recv %x, %s", s.Codec, s.Data)
	}

	if s, err := client.SimpleRead(); err != nil {
		t.Errorf("client read %s", err.Error())
	} else {
		t.Logf("recv %x, %s", s.Codec, s.Data)
	}
}
