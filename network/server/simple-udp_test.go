package server

import (
	"context"
	"go-common/utils"
	"net"
	"testing"
	"time"
)

func TestSimpleUDP(t *testing.T) {
	server := NewSimpleUDPServer("0.0.0.0:99", utils.NewDefaultLogger())
	a := SimpleUDPHandleFunc(func(conn SimplePacketConn, addr net.Addr, data *SimpleData) {
		time.Sleep(1 * time.Second)
		conn.SimpleWrite(addr, data.Codec, data.MessageID, append([]byte("pong: ")))
	})
	server.RegisterCodec(0xc1c2, a)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		server.Run(ctx)
	}()
	time.Sleep(time.Second)
	defer cancel()

	client, err := NewSimpleUDPClient("127.0.0.1:99", utils.NewDefaultLogger())
	if err != nil {
		t.Errorf("client connect error: %s", err.Error())
	}

	if _, err = client.SimpleWrite(0xc1c2, 0x1, []byte("ping1")); err != nil {
		t.Errorf("client send error: %s", err.Error())
	}

	if _, err = client.SimpleWrite(0xc1c2, 0x2, []byte("ping2")); err != nil {
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
