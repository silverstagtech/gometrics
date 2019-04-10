package measurements

import (
	"sync"
)

// Gauge is used to keep track of a value. Gauges can be positive or negative.
// Gauge has no rate limit as it is intended to keep track of something so we can't
// not run it.
// Gauge makes use of a RWMutex to stop multiple calls from overriding the value.
// However they make use of int64 values and always start at 0.
// They can be set to a value or incremented and decremented at will.
type Gauge struct {
	Prefix string
	Name   string
	Value  int64
	Tags   map[string]string
	sync.RWMutex
}

// NewGauge returns a new gauge that can be used to track a value
func NewGauge(prefix, name string, tags map[string]string) *Gauge {
	return &Gauge{
		Prefix: prefix,
		Name:   name,
		Tags:   tags,
		Value:  int64(0),
	}
}

func (g *Gauge) Reset() {
	g.Lock()
	g.Value = 0
	g.Unlock()
}

func (g *Gauge) Increment() {
	g.Lock()
	g.Value++
	g.Unlock()
}

func (g *Gauge) Decrement() {
	g.Lock()
	g.Value--
	g.Unlock()
}

func (g *Gauge) Set(value int64) {
	g.Lock()
	g.Value = value
	g.Unlock()
}

// Export is used to take a snapshot in time value of the gauge.
// It would be used by the serializers to read out the value.
// Because gauges are long lived they come with a mutex which can't be copied.
func (g *Gauge) Export() *Gauge {
	g.RLock()
	copyTags := make(map[string]string)
	for key, value := range g.Tags {
		copyTags[key] = value
	}

	copy := &Gauge{
		Prefix: g.Prefix,
		Name:   g.Name,
		Value:  g.Value,
		Tags:   copyTags,
	}
	g.RUnlock()

	return copy
}
