package streamer

import "fmt"

var (
	errTooLarge tooLarge = fmt.Errorf("datagram too large for packet buffer")
	errOverCap  overCap  = fmt.Errorf("datagram will make buffer too large")
)

type tooLarge error
type overCap error

type packet struct {
	maxSize int
	body    []byte
	index   int
	joint   []byte
}

func newPacket(maxSize int, joint []byte) *packet {
	p := &packet{
		maxSize: maxSize,
		joint:   joint,
		body:    make([]byte, maxSize),
	}
	p.reset()
	return p
}

// add will see if the bytes given will fit. If not it will return a errPacketTooLarge.
// At this point you can read from the packet which will reset it and try again.
func (p *packet) add(bs []byte) error {
	// Will bs ever fit?
	if p.maxSize+len(p.joint) > len(bs) {
		return errTooLarge
	}
	// Will bs fit in whats left?
	if (len(bs) + (p.index + 1) + len(p.joint)) > p.maxSize {
		return errOverCap
	}
	// Add the joining bytes if its not the first data in the buffer.
	if p.index != 0 {
		for _, b := range p.joint {
			p.body[p.index] = b
			p.index++
		}
	}
	// Add the data
	for _, b := range bs {
		p.body[p.index] = b
		p.index++
	}
	return nil
}

// read will return the data in the buffer and then reset it.
// It is possible to get a read of zero bytes if there is noting
func (p *packet) read() []byte {
	end := 0
	for index, b := range p.body {
		if b == EOF {
			end = index
		}
	}
	out := make([]byte, end)
	if end != 0 {
		copy(out, p.body[:end])
	}
	p.reset()
	return out
}

func (p *packet) reset() {
	for index := range p.body {
		p.body[index] = EOF
	}
	p.index = 0
}
