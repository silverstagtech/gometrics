package gometrics

import (
	"math/rand"
	"time"

	"github.com/silverstagtech/gometrics/measurements"
	"github.com/silverstagtech/gometrics/serializers"
	"github.com/silverstagtech/gometrics/shippers"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MetricFactory is used to create and send metrics out.
// It will serialize metrics and send them to the shipper that is chosen by the user.
//
// Metric factories are safe for concurrent usage.
//
// Metric Factory uses the rate on measurements to determine if the measurement should be sent to the shipper.
// Rates 1 or greater will always be sent. Anything below is used as a percentage of a chance.
// ie. 0.5 is 50% of the time, 0.25 is 25% and 0.0003 is 0.03%
type MetricFactory struct {
	prefix      string
	defaultTags map[string]string
	//  gauges map[string]*Gauge
	serializer serializers.Serializer
	shipper    shippers.Shipper
	onError    func(error)
}

// NewFactory will return a pointer to a MetricFactory.
func NewFactory(prefix string, defaultTags map[string]string, serializer serializers.Serializer, shipper shippers.Shipper, onError func(error)) *MetricFactory {
	return &MetricFactory{
		prefix:      prefix,
		defaultTags: defaultTags,
		serializer:  serializer,
		shipper:     shipper,
		onError:     onError,
	}
}

// Counter is used to create a new event measurements.
func (mf *MetricFactory) Counter(name string, value int, rate float32, tags map[string]string) {
	if !gatekeeper(rate) {
		return
	}
	b, err := mf.serializer.Counter(
		measurements.NewCounter(mf.prefix, name, value, rate, mergeTags(mf.defaultTags, tags)),
	)
	if err != nil {
		mf.onError(err)
		return
	}
	mf.shipper.Ship(b)

}

//func (mf *MetricFactory) Timer()   {}
//func (mf *MetricFactory) Poly()    {}
//
//func (mf *MetricFactory) Gauge() {}
