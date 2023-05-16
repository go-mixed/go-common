package chanUtils

// From: https://colobu.com/2021/05/11/unbounded-channel-in-go/

import (
	"context"
	"gopkg.in/go-mixed/go-common.v1/utils/list"
	"sync/atomic"
)

// UnboundedChan is an unbounded chan.
// In is used to write without blocking, which supports multiple writers.
// and Out is used to read, which supports multiple readers.
// You can close the In channel if you want.
//
// UnboundedChan 是一个无边界的chan，In用于写入（不会阻塞），Out用于读取，支持多个写入者和多个读取者。
// 关闭In后，Out会读取完缓存中的数据后自动关闭。
type UnboundedChan[T any] struct {
	bufCount atomic.Int64
	In       chan<- T // public channel for write
	Out      <-chan T // public channel for read
	in       chan T
	out      chan T
	buffer   *listUtils.RingBuffer[T] // buffer

	clearCancel context.CancelFunc
}

// NewUnboundedChan creates the unbounded chan.
// in is used to write without blocking, which supports multiple writers.
// and out is used to read, which supports multiple readers.
// You can close the in channel if you want.
//
// 创建无边界Channel，输入初始化容量（会自动扩容）
func NewUnboundedChan[T any](initCapacity int) *UnboundedChan[T] {
	return NewUnboundedChanSize[T](context.Background(), initCapacity, initCapacity, initCapacity)
}

// NewUnboundedChanContext creates the unbounded chan with context.
// in is used to write without blocking, which supports multiple writers.
// and out is used to read, which supports multiple readers.
// You can close the in channel if you want.
//
// 创建无边界Channel，输入初始化容量（会自动扩容），并指定Context
func NewUnboundedChanContext[T any](ctx context.Context, initCapacity int) *UnboundedChan[T] {
	return NewUnboundedChanSize[T](ctx, initCapacity, initCapacity, initCapacity)
}

// NewUnboundedChanSize is like NewUnboundedChan, but you can set initial capacity for In, Out, Buffer.
func NewUnboundedChanSize[T any](ctx context.Context, initInCapacity, initOutCapacity, initBufCapacity int) *UnboundedChan[T] {
	in := make(chan T, initInCapacity)
	out := make(chan T, initOutCapacity)
	ch := &UnboundedChan[T]{In: in, Out: out, in: in, out: out, buffer: listUtils.NewRingBuffer[T](initBufCapacity), bufCount: atomic.Int64{}}

	go ch.processing(ctx)

	return ch
}

// processing the main coroutine for processing data.
func (ch *UnboundedChan[T]) processing(ctx context.Context) {
	defer close(ch.out)
	var clearCtx context.Context

	// initialize clearCtx and clearCancel
	clearCtx, ch.clearCancel = context.WithCancel(ctx)
	defer ch.clearCancel()

	// drain buffer to out, and reset buffer.
	// 消耗缓存到out，并清空缓存
	drain := func() {
		for !ch.buffer.IsEmpty() {
			ch.out <- ch.buffer.Pop()
			ch.bufCount.Add(-1)
		}

		ch.buffer.Reset()
		ch.bufCount.Store(0)
	}

	// clear buffer and in, out. coroutine safe because of clearAll is running in processing coroutine.
	// 清空缓存和in、out, 协程安全（因为clearAll在processing协程中运行）
	clearAll := func() {
		ch.buffer.Reset()
		ch.bufCount.Store(0)
		ClearChan[T](ch.in)
		ClearChan[T](ch.out)

		// Prevent memory leaks
		ch.clearCancel()
		// re-define clearCtx and clearCancel,
		clearCtx, ch.clearCancel = context.WithCancel(ctx)
	}

	// main loop
	for {
		select {
		case <-ctx.Done():
			return
		case <-clearCtx.Done():
			clearAll()
		case val, ok := <-ch.in:
			if !ok { // in is closed
				drain()
				return
			}

			// make sure values' order
			// buffer has some values
			if ch.bufCount.Load() > 0 {
				ch.buffer.Write(val)
				ch.bufCount.Add(1)
			} else {
				// out is not full
				select {
				case ch.out <- val:
					continue
				default:
				}

				// out is full
				ch.buffer.Write(val)
				ch.bufCount.Add(1)
			}

			for !ch.buffer.IsEmpty() {
				select {
				case <-ctx.Done():
					return
				case <-clearCtx.Done():
					clearAll()
				case val, ok = <-ch.in:
					if !ok { // in is closed
						drain()
						return
					}
					ch.buffer.Write(val)
					ch.bufCount.Add(1)
				case ch.out <- ch.buffer.Peek():
					ch.buffer.Pop()
					ch.bufCount.Add(-1)
					if ch.buffer.IsEmpty() && ch.buffer.Capacity() > ch.buffer.InitialSize() { // after burst
						ch.buffer.Reset()
						ch.bufCount.Store(0)
					}
				}
			}
		}
	}
}

// Produce writes a value to the in channel without not block.
//
// 写入In通道，不会阻塞
func (ch *UnboundedChan[T]) Produce(val T) {
	ch.In <- val
}

// Len returns len of In plus len of Out plus len of buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
//
// 通道长度，返回In+Out+buffer的长度，不精确，只是用于评估通道中元素的大致数量。
func (ch *UnboundedChan[T]) Len() int {
	return len(ch.in) + ch.BufLen() + len(ch.out)
}

// BufLen returns len of the buffer.
// It is not accurate and only for your evaluating approximate number of elements in this chan,
// see https://github.com/smallnest/chanx/issues/7.
//
// 缓冲长度，不精确，只是用于评估通道中元素的大致数量。
func (ch *UnboundedChan[T]) BufLen() int {
	return int(ch.bufCount.Load())
}

// Close closes the In channel.
// It will close the Out channel after all elements in the buffer are read.
//
// 关闭In通道，Out会消耗完缓冲中的元素后关闭。
func (ch *UnboundedChan[T]) Close() {
	close(ch.in)
}

// Clear the channels of in, out, and buffer
//
// 清空in、out、buffer
func (ch *UnboundedChan[T]) Clear() {
	if ch.clearCancel != nil {
		ch.clearCancel()
	}
}
