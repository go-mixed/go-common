package listUtils

// From: https://github.com/smallnest/chanx/blob/main/ringbuffer.go

import "github.com/pkg/errors"

var ErrRingBufferIsEmpty = errors.New("ring buffer is empty")

// RingBuffer is a ring buffer for common types.
// It is never full and always grows if it will be full.
// It is not thread-safe(goroutine-safe) so you must use the lock-like synchronization primitive to use it in multiple writers and multiple readers.
//
// RingBuffer是一个通用类型的环形缓冲列表。
// 当读取比写入多时，无需扩容，会环形复用之前的空间。当写入比读取多时，会自动扩容。
// 它不是线程安全的，在并发场合必须使用锁的来控制写入和读取。
type RingBuffer[T any] struct {
	buf         []T
	initialSize int
	size        int
	r           int // read pointer
	w           int // write pointer
}

// NewRingBuffer creates a new ring buffer. initialSize must be greater than 1.
//
// 创建一个新的环形缓冲列表。 initialSize必须大于1。
func NewRingBuffer[T any](initialSize int) *RingBuffer[T] {
	if initialSize <= 0 {
		panic("initial size must be great than zero")
	}
	// initial size must >= 2
	if initialSize == 1 {
		initialSize = 2
	}

	return &RingBuffer[T]{
		buf:         make([]T, initialSize),
		initialSize: initialSize,
		size:        initialSize,
	}
}

// Read the first element from the ring buffer, and remove it.
//
// 从环形缓冲列表中读取第一个元素，并删除它。
func (r *RingBuffer[T]) Read() (T, error) {
	var t T
	if r.r == r.w {
		return t, ErrRingBufferIsEmpty
	}

	v := r.buf[r.r]
	r.r++
	if r.r == r.size {
		r.r = 0
	}

	return v, nil
}

// Pop read the first element, and remove it.
//
// 读取第一个元素，并删除它。
func (r *RingBuffer[T]) Pop() T {
	v, err := r.Read()
	if err == ErrRingBufferIsEmpty { // Empty
		panic(ErrRingBufferIsEmpty.Error())
	}

	return v
}

// Peek returns the first element from the ring buffer without removing it.
//
// 从环形缓冲列表中返回第一个元素，而不删除它。
func (r *RingBuffer[T]) Peek() T {
	if r.r == r.w { // Empty
		panic(ErrRingBufferIsEmpty.Error())
	}

	v := r.buf[r.r]
	return v
}

// Write appends a new element to the ring buffer.
//
// 将一个新元素添加到环形缓冲列表中。
func (r *RingBuffer[T]) Write(v T) {
	r.buf[r.w] = v
	r.w++

	if r.w == r.size {
		r.w = 0
	}

	if r.w == r.r { // full
		r.grow()
	}
}

// grows the ring buffer to a new size.
//
// 将环形缓冲列表扩展到一个新的大小。
func (r *RingBuffer[T]) grow() {
	var size int
	if r.size < 1024 {
		size = r.size * 2
	} else {
		size = r.size + r.size/4
	}

	buf := make([]T, size)

	copy(buf[0:], r.buf[r.r:])
	copy(buf[r.size-r.r:], r.buf[0:r.r])

	r.r = 0
	r.w = r.size
	r.size = size
	r.buf = buf
}

// IsEmpty returns true if the ring buffer is empty.
//
// 如果环形缓冲列表为空，则返回true。
func (r *RingBuffer[T]) IsEmpty() bool {
	return r.r == r.w
}

// InitialSize returns the initial size of the ring buffer.
//
// 返回环形缓冲列表的初始大小。
func (r *RingBuffer[T]) InitialSize() int {
	return r.initialSize
}

// Capacity returns the size of the underlying buffer.
//
// 返回底层缓冲区的容量。
func (r *RingBuffer[T]) Capacity() int {
	return r.size
}

// Len returns the number of elements in the ring buffer.
//
// 返回环形缓冲列表中的元素数量。
func (r *RingBuffer[T]) Len() int {
	if r.r == r.w {
		return 0
	}

	if r.w > r.r {
		return r.w - r.r
	}

	return r.size - r.r + r.w
}

// Reset resets the ring buffer.
//
// 重置环形缓冲列表。
func (r *RingBuffer[T]) Reset() {
	r.r = 0
	r.w = 0
	r.size = r.initialSize
	r.buf = make([]T, r.initialSize)
}
