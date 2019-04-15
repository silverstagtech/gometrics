package streamer

type errTooLarge struct {
	s string
}

func (errTooLarge) Error() string {
	return "datagram too large for packet buffer"
}

type errOverCap struct {
	s string
}

func (errOverCap) Error() string {
	return "datagram will make buffer too large"
}

type packet struct {
	maxSize int
	body    []byte
	index   int
}

func newPacket(maxSize int) *packet {
	p := &packet{
		maxSize: maxSize,
		body:    make([]byte, maxSize),
	}
	p.reset()
	return p
}

// add will see if the bytes given will fit. If not it will return a errPacketTooLarge.
// At this point you can read from the packet which will reset it and try again.
func (p *packet) add(bs []byte) error {
	// Will bs ever fit?
	if len(bs) > p.maxSize {
		return errTooLarge{}
	}
	// Will bs fit in whats left?
	if (len(bs) + (p.index + 1)) > p.maxSize {
		return errOverCap{}
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
			break
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
