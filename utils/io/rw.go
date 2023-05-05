package ioUtils

import "io"

type wrapperWriter struct {
	fn func(p []byte) (n int, err error)
}

type wrapperReader struct {
	fn func(p []byte) (n int, err error)
}

var _ io.Writer = (*wrapperWriter)(nil)
var _ io.Reader = (*wrapperReader)(nil)

func NewWrapperWriter(fn func(p []byte) (n int, err error)) io.Writer {
	return &wrapperWriter{fn: fn}
}

func (w *wrapperWriter) Write(p []byte) (n int, err error) {
	return w.fn(p)
}

func NewWrapperReader(fn func(p []byte) (n int, err error)) io.Reader {
	return &wrapperReader{fn: fn}
}

func (w *wrapperReader) Read(p []byte) (n int, err error) {
	return w.fn(p)
}
