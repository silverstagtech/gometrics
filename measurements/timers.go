package measurements

import "time"

// Timer is used to measure how long a procedure took.
// It could be a download, write to a DB or how long a function took.
type Timer struct {
	Prefix string
	Name   string
	Value  time.Duration
	Unit   string
	Tags   map[string]string
}

// NewTimer returns a pointer to a Timer
func NewTimer(prefix, name string, value time.Duration, unit string, tags map[string]string) *Timer {
	return &Timer{
		Prefix: prefix,
		Name:   name,
		Value:  value,
		Tags:   tags,
		Unit:   unit,
	}
}
