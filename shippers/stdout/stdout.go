package stdout

import "fmt"

// StdOut is used to print metrics to STDOUT via fmt.Println
type StdOut struct {
	onErr func(error) bool
}

// New returns a pointer to a StdOut
func New() *StdOut {
	return &StdOut{
		onErr: func(error) bool { return true },
	}
}

// Ship prints the bytes as a string to StdOut via fmt.Println
func (StdOut) Ship(b []byte) {
	fmt.Println(string(b))
}

// Shutdown returns a closed channel as there is nothing to shutdown
func (StdOut) Shutdown() chan struct{} {
	c := make(chan struct{}, 1)
	close(c)
	return c
}
