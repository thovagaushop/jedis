package ringbuffer

import (
	"errors"
	"sync/atomic"
)

type RingBuffer[T any] struct {
	head   uint64
	_      [56]byte // padding
	tail   uint64
	_      [56]byte // padding
	mask   uint64
	buffer []T
}

func NewRingBuffer[T any](size uint64) (*RingBuffer[T], error) {
	if size&(size-1) != 0 {
		return nil, errors.New("size must be a power of 2")
	}

	return &RingBuffer[T]{
		mask:   size - 1,
		buffer: make([]T, size),
	}, nil
}

func (rb *RingBuffer[T]) Push(value T) bool {
	tail := rb.tail
	head := atomic.LoadUint64(&rb.head)

	if tail-head >= uint64(len(rb.buffer)) {
		return false
	}

	rb.buffer[tail&rb.mask] = value
	atomic.StoreUint64(&rb.tail, tail+1)
	return true
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
	var zero T
	tail := atomic.LoadUint64(&rb.tail)
	head := rb.head

	if head == tail {
		return zero, false
	}

	value := rb.buffer[head&rb.mask]
	atomic.StoreUint64(&rb.head, head+1)
	return value, true
}
