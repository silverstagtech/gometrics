package silent

import (
	"github.com/silverstagtech/gometrics/measurements"
)

// Silent is a struct that just consumes requests and returns nothing.
// Mainly used for testing or initial values of GlobalMetricFactory
type Silent struct{}

// New returns a pointer to a Silent struct
func New() *Silent {
	return &Silent{}
}

// Counter returns an empty []byte and a nil error
func (s Silent) Counter(m *measurements.Counter) ([]byte, error) { return []byte{}, nil }

// Timer  returns an empty []byte and a nil error
func (s Silent) Timer(m *measurements.Timer) ([]byte, error) { return []byte{}, nil }

// Gauge  returns an empty []byte and a nil error
func (s Silent) Gauge(m *measurements.Gauge) ([]byte, error) { return []byte{}, nil }

// Poly  returns an empty []byte and a nil error
func (s Silent) Poly(m *measurements.Poly) ([]byte, error) { return []byte{}, nil }
