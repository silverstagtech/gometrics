package gometrics

import (
	"math/rand"
	"sync"
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
	prefix            string
	defaultTags       map[string]string
	registeredGauges  map[string]*measurements.Gauge
	serializer        serializers.Serializer
	shipper           shippers.Shipper
	onError           func(error)
	flushGaugesTicker *simpleTicker
	mutex             sync.RWMutex
}

// NewFactory will return a pointer to a MetricFactory.
// See MetricFactory for more details.
func NewFactory(prefix string, defaultTags map[string]string, serializer serializers.Serializer, shipper shippers.Shipper, onError func(error)) *MetricFactory {
	return &MetricFactory{
		prefix:      prefix,
		defaultTags: defaultTags,
		serializer:  serializer,
		shipper:     shipper,
		onError:     onError,
	}
}

// FlushGaugesEvery takes an int which represents milliseconds ans is used to flush gauges when required.
// As gauges track state they do not need to be flush out as often as counters.
// If you are going to be using gauges then you will need to set this, if not then omit it.
func (mf *MetricFactory) FlushGaugesEvery(milliseconds int) {
	mf.flushGaugesTicker = newSimpleTicker(milliseconds)
	mf.flushGaugesTicker.start()
	go func() {
		for {
			select {
			case _, ok := <-mf.flushGaugesTicker.c:
				if !ok {
					return
				}
				mf.flushGauges()
			}
		}
	}()
}

func (mf *MetricFactory) flushGauges() {
	mf.mutex.RLock()
	defer mf.mutex.RUnlock()

	if mf.registeredGauges == nil {
		return
	}
	for _, protectedGauge := range mf.registeredGauges {
		b, err := mf.serializer.Gauge(protectedGauge.Export())
		if err != nil {
			mf.onError(err)
			continue
		}
		mf.shipper.Ship(b)
	}

}

// Shutdown tears down the factory and tells the shippers to flush anything they have left
// in the buffers if they have buffers.
func (mf *MetricFactory) Shutdown() chan struct{} {
	// Stop the ticker if it is there and flush gauges for the last time.
	if mf.flushGaugesTicker != nil {
		mf.flushGaugesTicker.stop()
		mf.flushGauges()
	}
	// Shutdown our shipper
	return mf.shipper.Shutdown()
}

// Counter is used to create a new event "counter" measurement.
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

// RegisterGauge setup the named gauge and sets it to zero.
// Gauges have to be registered and are flushed out using the FlushGaugeEvery(milliseconds) function.
// Because they track state they need to be long lived.
// If you need to use Gauges remember to call the FlushGaugeEvery(milliseconds) function when creating
// the MetricFactory.
func (mf *MetricFactory) RegisterGauge(name string, tags map[string]string) {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()
	if mf.registeredGauges == nil {
		mf.registeredGauges = make(map[string]*measurements.Gauge)
	}
	if _, ok := mf.registeredGauges[name]; !ok {
		mf.registeredGauges[name] = measurements.NewGauge(mf.prefix, name, mergeTags(mf.defaultTags, tags))
	}
}

// RemoveGauge allows you to stop a gauge.
// Its not called DeregisterGauge because that name would be too close to Register
func (mf *MetricFactory) RemoveGauge(name string) {
	mf.mutex.Lock()
	defer mf.mutex.Unlock()

	if mf.registeredGauges == nil {
		return
	}
	if _, ok := mf.registeredGauges[name]; !ok {
		delete(mf.registeredGauges, name)
	}
}

// GaugeInc is used to increment the named gauge.
func (mf *MetricFactory) GaugeInc(name string) {
	if _, ok := mf.registeredGauges[name]; !ok {
		return
	}
	mf.registeredGauges[name].Increment()
}

// GaugeDec is used to decrement the named gauge.
func (mf *MetricFactory) GaugeDec(name string) {
	if _, ok := mf.registeredGauges[name]; !ok {
		return
	}
	mf.registeredGauges[name].Decrement()
}

// GaugeSet is used to set the value of a gauge to a desired value.
func (mf *MetricFactory) GaugeSet(name string, value int64) {
	if _, ok := mf.registeredGauges[name]; !ok {
		return
	}
	mf.registeredGauges[name].Set(value)
}

// Timer takes a details of a timer and send it to the shipper.
// You need to tell the timer what precision you want to show in your measurement.
// The value can and most likely will be from time.Since().
func (mf *MetricFactory) Timer(name string, precision TimerPrecision, value time.Duration, tags map[string]string) {
	divider, unit := timerUnit(precision)
	duration := value / divider
	timer := measurements.NewTimer(mf.prefix, name, duration, unit, mergeTags(mf.defaultTags, tags))
	b, err := mf.serializer.Timer(timer)
	if err != nil {
		mf.onError(err)
		return
	}
	mf.shipper.Ship(b)
}

func (mf *MetricFactory) NewPoly(name string, fields map[string]interface{}, tags map[string]string) *measurements.Poly {
	return measurements.NewPoly(mf.prefix, name, fields, mergeTags(mf.defaultTags, tags))
}

func (mf *MetricFactory) SubmitPoly(poly *measurements.Poly) {
	b, err := mf.serializer.Poly(poly)
	if err != nil {
		mf.onError(err)
		return
	}

	mf.shipper.Ship(b)
}
