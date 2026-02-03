package ringbuffer

import "sync"

// RingBuffer is a concurrent-safe ring buffer for int16 samples.
type RingBuffer struct {
	buf        []int16
	size       int
	readIndex  int
	writeIndex int
	closed     bool
	mu         sync.Mutex
	cond       *sync.Cond
}

// New creates a new RingBuffer of a given size.
func New(size int) *RingBuffer {
	rb := &RingBuffer{
		buf:  make([]int16, size),
		size: size,
	}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

// AvailableWrite returns the number of samples that can be written to the buffer.
func (rb *RingBuffer) AvailableWrite() int {
	if rb.writeIndex >= rb.readIndex {
		return rb.size - (rb.writeIndex - rb.readIndex) - 1
	}
	return rb.readIndex - rb.writeIndex - 1
}

// AvailableRead returns the number of samples available for reading.
func (rb *RingBuffer) AvailableRead() int {
	if rb.writeIndex >= rb.readIndex {
		return rb.writeIndex - rb.readIndex
	}
	return rb.size - rb.readIndex + rb.writeIndex
}

// Close marks the buffer as closed, indicating no more writes will occur.
// It broadcasts to all waiting readers to wake them up.
func (rb *RingBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.closed = true
	rb.cond.Broadcast() // Wake up any readers waiting for data.
}

// Write adds data to the buffer, blocking until space is available.
func (rb *RingBuffer) Write(data []int16) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.closed {
		// Or return an error, but for this use case, panicking is acceptable
		// as it indicates a programming error.
		panic("write to closed ring buffer")
	}

	n := len(data)
	for i := 0; i < n; {
		// Wait for space to become available.
		for rb.AvailableWrite() == 0 {
			rb.cond.Wait()
		}

		// Copy in one or two chunks.
		if rb.writeIndex >= rb.readIndex {
			// Write up to the end of the buffer.
			written := copy(rb.buf[rb.writeIndex:], data[i:])
			rb.writeIndex = (rb.writeIndex + written) % rb.size
			i += written
		} else {
			// Write up to the read index.
			written := copy(rb.buf[rb.writeIndex:rb.readIndex-1], data[i:])
			rb.writeIndex += written
			i += written
		}
		rb.cond.Broadcast() // Signal reader that data is available.
	}
}

// Read retrieves n samples from the buffer, blocking until they are available.
// If the buffer is closed and no more data is available, it returns nil.
func (rb *RingBuffer) Read(n int) []int16 {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	// Wait for data, but stop waiting if the buffer is closed.
	// The reader should wait as long as the buffer doesn't have enough data AND it's not closed.
	// Once closed, the reader should proceed to read whatever is left.
	for !rb.closed && rb.AvailableRead() < n {
		rb.cond.Wait()
	}

	// If the buffer is closed and empty, it's the end of the stream.
	if rb.closed && rb.AvailableRead() == 0 {
		return nil
	}

	// Read what's available, up to a maximum of n samples.
	readSize := n
	if rb.AvailableRead() < readSize {
		readSize = rb.AvailableRead()
	}

	if readSize == 0 {
		return nil
	}

	data := make([]int16, readSize)
	if rb.readIndex+readSize <= rb.size {
		copy(data, rb.buf[rb.readIndex:rb.readIndex+readSize])
	} else {
		part1 := rb.size - rb.readIndex
		copy(data, rb.buf[rb.readIndex:])
		copy(data[part1:], rb.buf[0:readSize-part1])
	}
	rb.readIndex = (rb.readIndex + readSize) % rb.size
	rb.cond.Broadcast()
	return data
}
