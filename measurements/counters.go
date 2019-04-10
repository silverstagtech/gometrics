package measurements

// Counter is used to create a measurement that signifies a single event which needs to be counted.
type Counter struct {
	Prefix string
	Name   string
	Value  int
	Rate   float32
	Tags   map[string]string
}

// NewCounter returns a pointer to Counter that can be sent to a serializer.
func NewCounter(prefix, name string, value int, rate float32, tags map[string]string) *Counter {
	return &Counter{
		Prefix: prefix,
		Name:   name,
		Value:  value,
		Rate:   rate,
		Tags:   tags,
	}
}
