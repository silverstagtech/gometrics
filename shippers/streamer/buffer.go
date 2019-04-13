package streamer

import (
	"sync"
)

type buffer struct {
	currentSize int
	buf         [][]byte
	in          chan []byte
	out         chan [][]byte
	maxSize     int
	trigger     chan struct{}
	sync.RWMutex
}

func newBuffer() *buffer {
	b := &buffer{
		maxSize: DefaultMaxBuffer,
		in:      make(chan []byte, 10),
		out:     make(chan [][]byte, 1),
		trigger: make(chan struct{}, 1),
	}
	b.reset()
	return b
}

func (buf *buffer) init() {
	go func() {
		for {
			select {
			case <-buf.trigger:
				buf.flush()
			case b, ok := <-buf.in:
				if !ok {
					// channel is closed we are not getting any more.
					buf.flush()
					close(buf.out)
					return
				}
				bytesSize := len(b)
				if buf.currentSize+bytesSize > buf.maxSize {
					buf.flush()
				}
				buf.Lock()
				buf.currentSize = buf.currentSize + bytesSize
				buf.buf = append(buf.buf, b)
				buf.Unlock()
			}
		}
	}()
}

func (buf *buffer) flush() {
	buf.Lock()
	out := make([][]byte, len(buf.buf))
	copy(out, buf.buf)
	buf.reset()
	buf.Unlock()

	buf.out <- out
}

func (buf *buffer) reset() {
	buf.currentSize = 0
	buf.buf = make([][]byte, 0)
}
