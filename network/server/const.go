package server

import "errors"

type Codec uint16

// ErrServerClosed is returned by the Server's Serve,
var ErrServerClosed = errors.New("server: server closed")
