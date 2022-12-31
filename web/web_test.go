package web

import (
	"context"
	"gopkg.in/go-mixed/go-common.v1/utils/conv"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

type httpHandle struct {
}

func (*httpHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func TestNewHttpServer(t *testing.T) {
	port := rand.Intn(50_000) + 10_000
	ctx, cancel := context.WithCancel(context.Background())
	server := NewHttpServer(DefaultServerOptions("127.0.0.1:" + conv.Itoa(port)))

	server.SetDefaultServeHandler(&httpHandle{}, nil)
	go func() {
		time.AfterFunc(2*time.Second, func() {
			cancel()
		})
	}()
	err := server.Run(ctx, nil)
	if err != nil {
		t.Errorf(err.Error())
	}
}
