package serializers

import (
	"github.com/silverstagtech/gometrics/measurements"
)

type Serializer interface {
	Counter(*measurements.Counter) ([]byte, error)
	Timer(*measurements.Timer) ([]byte, error)
	Gauge(*measurements.Gauge) ([]byte, error)
	Poly(*measurements.Poly) ([]byte, error)
}
