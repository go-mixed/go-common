package web

import (
	"net/http"
	"testing"
)

func TestMiddleware(t *testing.T) {
	server := NewHttpServer(DefaultServerOptions("0.0.0.0:1234"))
	server.Use(func(w http.ResponseWriter, r *http.Request, nextHandler http.Handler) {
		t.Logf("----%s----%s---1\n", r.RemoteAddr, r.RequestURI)
		nextHandler.ServeHTTP(w, r)
	}, func(w http.ResponseWriter, r *http.Request, nextHandler http.Handler) {
		r.RequestURI += "/123"
		t.Logf("----%s----%s---2\n", r.RemoteAddr, r.RequestURI)
		w.Write([]byte("abc"))
		nextHandler.ServeHTTP(w, r)

	}, func(w http.ResponseWriter, r *http.Request, nextHandler http.Handler) {
		r.RequestURI += "/456"
		nextHandler.ServeHTTP(w, r)
		t.Logf("----%s----%s---4\n", r.RemoteAddr, r.RequestURI)
		w.Write([]byte("hgg"))

	})
	server.SetDefaultServeHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("----%s----%s---3\n", r.RemoteAddr, r.RequestURI)
		w.Write([]byte("def"))

	}), nil)

	go func() {
		server.Run(nil, nil)
	}()

	go func() {
		http.Get("http://127.0.0.1:1234/")
		server.Close()
	}()
}
