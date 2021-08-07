package web

import (
	"go-common/utils/conv"
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
	stopChan := make(chan bool)
	server := NewHttpServer(DefaultServerOptions("127.0.0.1:" + conv.Itoa(port)))

	server.SetDefaultServeHandler(&httpHandle{}, nil)
	go func() {
		time.AfterFunc(2*time.Second, func() {
			close(stopChan)
		})
	}()
	_, err := server.Run(stopChan)
	if err != nil {
		t.Errorf(err.Error())
	}
}
