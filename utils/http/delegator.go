// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http_utils

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

const (
	closeNotifier = 1 << iota
	flusher
	hijacker
	readerFrom
	pusher
)

type WriterDelegator interface {
	http.ResponseWriter

	Status() int
	Written() int64
}

type ResponseWriterDelegator struct {
	http.ResponseWriter

	status             int
	written            int64
	wroteHeader        bool
	observeWriteHeader func(int)
}

func (r *ResponseWriterDelegator) Status() int {
	return r.status
}

func (r *ResponseWriterDelegator) Written() int64 {
	return r.written
}

func (r *ResponseWriterDelegator) WriteHeader(code int) {
	if r.observeWriteHeader != nil && !r.wroteHeader {
		// Only call observeWriteHeader for the 1st time. It's a bug if
		// WriteHeader is called more than once, but we want to protect
		// against it here. Note that we still delegate the WriteHeader
		// to the original ResponseWriter to not mask the bug from it.
		r.observeWriteHeader(code)
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseWriterDelegator) Write(b []byte) (int, error) {
	// If applicable, call WriteHeader here so that observeWriteHeader is
	// handled appropriately.
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.written += int64(n)
	return n, err
}

type CloseNotifierDelegator struct{ *ResponseWriterDelegator }
type FlusherDelegator struct{ *ResponseWriterDelegator }
type HijackerDelegator struct{ *ResponseWriterDelegator }
type ReaderFromDelegator struct{ *ResponseWriterDelegator }
type PusherDelegator struct{ *ResponseWriterDelegator }

func (d CloseNotifierDelegator) CloseNotify() <-chan bool {
	//nolint:staticcheck // Ignore SA1019. http.CloseNotifier is deprecated but we keep it here to not break existing users.
	return d.ResponseWriter.(http.CloseNotifier).CloseNotify()
}
func (d FlusherDelegator) Flush() {
	// If applicable, call WriteHeader here so that observeWriteHeader is
	// handled appropriately.
	if !d.wroteHeader {
		d.WriteHeader(http.StatusOK)
	}
	d.ResponseWriter.(http.Flusher).Flush()
}
func (d HijackerDelegator) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return d.ResponseWriter.(http.Hijacker).Hijack()
}
func (d ReaderFromDelegator) ReadFrom(re io.Reader) (int64, error) {
	// If applicable, call WriteHeader here so that observeWriteHeader is
	// handled appropriately.
	if !d.wroteHeader {
		d.WriteHeader(http.StatusOK)
	}
	n, err := d.ResponseWriter.(io.ReaderFrom).ReadFrom(re)
	d.written += n
	return n, err
}
func (d PusherDelegator) Push(target string, opts *http.PushOptions) error {
	return d.ResponseWriter.(http.Pusher).Push(target, opts)
}

var pickDelegator = make([]func(*ResponseWriterDelegator) WriterDelegator, 32)

func init() {
	// TODO(beorn7): Code generation would help here.
	pickDelegator[0] = func(d *ResponseWriterDelegator) WriterDelegator { // 0
		return d
	}
	pickDelegator[closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 1
		return CloseNotifierDelegator{d}
	}
	pickDelegator[flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 2
		return FlusherDelegator{d}
	}
	pickDelegator[flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 3
		return struct {
			*ResponseWriterDelegator
			http.Flusher
			http.CloseNotifier
		}{d, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[hijacker] = func(d *ResponseWriterDelegator) WriterDelegator { // 4
		return HijackerDelegator{d}
	}
	pickDelegator[hijacker+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 5
		return struct {
			*ResponseWriterDelegator
			http.Hijacker
			http.CloseNotifier
		}{d, HijackerDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[hijacker+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 6
		return struct {
			*ResponseWriterDelegator
			http.Hijacker
			http.Flusher
		}{d, HijackerDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[hijacker+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 7
		return struct {
			*ResponseWriterDelegator
			http.Hijacker
			http.Flusher
			http.CloseNotifier
		}{d, HijackerDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[readerFrom] = func(d *ResponseWriterDelegator) WriterDelegator { // 8
		return ReaderFromDelegator{d}
	}
	pickDelegator[readerFrom+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 9
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.CloseNotifier
		}{d, ReaderFromDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[readerFrom+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 10
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Flusher
		}{d, ReaderFromDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[readerFrom+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 11
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Flusher
			http.CloseNotifier
		}{d, ReaderFromDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[readerFrom+hijacker] = func(d *ResponseWriterDelegator) WriterDelegator { // 12
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Hijacker
		}{d, ReaderFromDelegator{d}, HijackerDelegator{d}}
	}
	pickDelegator[readerFrom+hijacker+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 13
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Hijacker
			http.CloseNotifier
		}{d, ReaderFromDelegator{d}, HijackerDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[readerFrom+hijacker+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 14
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
		}{d, ReaderFromDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[readerFrom+hijacker+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 15
		return struct {
			*ResponseWriterDelegator
			io.ReaderFrom
			http.Hijacker
			http.Flusher
			http.CloseNotifier
		}{d, ReaderFromDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 16
		return PusherDelegator{d}
	}
	pickDelegator[pusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 17
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.CloseNotifier
		}{d, PusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 18
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Flusher
		}{d, PusherDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[pusher+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 19
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Flusher
			http.CloseNotifier
		}{d, PusherDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+hijacker] = func(d *ResponseWriterDelegator) WriterDelegator { // 20
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Hijacker
		}{d, PusherDelegator{d}, HijackerDelegator{d}}
	}
	pickDelegator[pusher+hijacker+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 21
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Hijacker
			http.CloseNotifier
		}{d, PusherDelegator{d}, HijackerDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+hijacker+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 22
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Hijacker
			http.Flusher
		}{d, PusherDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[pusher+hijacker+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { //23
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			http.Hijacker
			http.Flusher
			http.CloseNotifier
		}{d, PusherDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+readerFrom] = func(d *ResponseWriterDelegator) WriterDelegator { // 24
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 25
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.CloseNotifier
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 26
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Flusher
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 27
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Flusher
			http.CloseNotifier
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+hijacker] = func(d *ResponseWriterDelegator) WriterDelegator { // 28
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Hijacker
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, HijackerDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+hijacker+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 29
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Hijacker
			http.CloseNotifier
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, HijackerDelegator{d}, CloseNotifierDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+hijacker+flusher] = func(d *ResponseWriterDelegator) WriterDelegator { // 30
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Hijacker
			http.Flusher
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}}
	}
	pickDelegator[pusher+readerFrom+hijacker+flusher+closeNotifier] = func(d *ResponseWriterDelegator) WriterDelegator { // 31
		return struct {
			*ResponseWriterDelegator
			http.Pusher
			io.ReaderFrom
			http.Hijacker
			http.Flusher
			http.CloseNotifier
		}{d, PusherDelegator{d}, ReaderFromDelegator{d}, HijackerDelegator{d}, FlusherDelegator{d}, CloseNotifierDelegator{d}}
	}
}

func NewWriterDelegator(w http.ResponseWriter, observeWriteHeaderFunc func(int)) WriterDelegator {
	d := &ResponseWriterDelegator{
		ResponseWriter:     w,
		observeWriteHeader: observeWriteHeaderFunc,
	}

	id := 0
	//nolint:staticcheck // Ignore SA1019. http.CloseNotifier is deprecated but we keep it here to not break existing users.
	if _, ok := w.(http.CloseNotifier); ok {
		id += closeNotifier
	}
	if _, ok := w.(http.Flusher); ok {
		id += flusher
	}
	if _, ok := w.(http.Hijacker); ok {
		id += hijacker
	}
	if _, ok := w.(io.ReaderFrom); ok {
		id += readerFrom
	}
	if _, ok := w.(http.Pusher); ok {
		id += pusher
	}

	return pickDelegator[id](d)
}

type RequestBodyReaderDelegator struct {
	io.ReadCloser
	readBytes int64
}

func NewBodyDelegator(body io.ReadCloser) *RequestBodyReaderDelegator {
	d := &RequestBodyReaderDelegator{
		ReadCloser: body,
	}
	return d
}

func (r RequestBodyReaderDelegator) Read(p []byte) (n int, err error) {
	n, err = r.ReadCloser.Read(p)
	if n >= 0 {
		r.readBytes += int64(n)
	}
	return n, err
}

func (r RequestBodyReaderDelegator) ReadBytes() int64 {
	return r.readBytes
}

func (r RequestBodyReaderDelegator) Close() error {
	return r.ReadCloser.Close()
}
