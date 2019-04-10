package devnull

// DevNull just swallows data sent to it.
// Useful for testing.
type DevNull struct{}

// New returns a pointer to a DevNull which will just swallow bytes sent to it.
// Only really useful for testing.
func New() *DevNull {
	return &DevNull{}
}

// Ship simply drops the metrics
func (DevNull) Ship([]byte) {}

// Shutdown returns a closed channel to signal that its flushed all metrics
func (DevNull) Shutdown() chan struct{} {
	c := make(chan struct{}, 1)
	close(c)
	return c
}
