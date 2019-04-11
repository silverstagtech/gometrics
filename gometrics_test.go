package gometrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/silverstagtech/gotracer"

	"github.com/silverstagtech/gometrics/measurements"
	"github.com/silverstagtech/gometrics/serializers/json"
	"github.com/silverstagtech/gometrics/serializers/statsd"
	"github.com/silverstagtech/gometrics/shippers/devnull"
)

type tracing struct {
	tracer *gotracer.Tracer
}

func (tr tracing) Ship(b []byte) {
	tr.tracer.Send(string(b))
}

// Remember to reset the tracer yourself.
func (tr tracing) Shutdown() chan struct{} {
	c := make(chan struct{}, 1)
	close(c)
	return c
}

func TestCounter(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}
	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.FailNow() })

	factory.Counter("counter", 100, 1, nil)
	// TODO: make this a actual test
	// Look for the name and the tags for default and supplied.
	t.Logf("%v", sh.tracer.Show())
}

func BenchmarkCounter(b *testing.B) {
	se := json.New(false)
	sh := devnull.New()
	defaultTags := map[string]string{
		"testone": "1",
	}
	factory := NewFactory("test", defaultTags, se, sh, func(error) { b.FailNow() })
	for n := 0; n < b.N; n++ {
		factory.Counter("counter", 100, 1, nil)
	}
}

func TestRate(t *testing.T) {
	se := json.New(false)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}
	var rate float32 = 0.5
	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	for n := 0; n < 100; n++ {
		factory.Counter("counter", 1, rate, nil)
	}

	if sh.tracer.Len() >= 100 {
		t.Logf("counter with rate %f still gave %d counters.", rate, sh.tracer.Len())
		t.Fail()
	} else {
		t.Logf("counter with rate %f gave %d counters.", rate, sh.tracer.Len())
	}
}

func TestSingleGaugeZero(t *testing.T) {
	se := json.New(false)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	factory.FlushGaugesEvery(10)
	factory.RegisterGauge("gauge", nil)
	factory.Shutdown()

	if sh.tracer.Len() == 0 {
		t.Logf("Gauge did not get flushed")
		t.Fail()
	}
}

func TestSingleGaugeFlush(t *testing.T) {
	se := json.New(false)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	factory.FlushGaugesEvery(2)
	factory.RegisterGauge("gauge", nil)
	factory.GaugeInc("test")
	time.Sleep(time.Millisecond * 4)
	factory.Shutdown()

	if sh.tracer.Len() < 2 {
		t.Logf("Gauge did not get flushed")
		t.Fail()
	}
	t.Logf("%s\n", sh.tracer.Show())
}

func TestSingleGauge(t *testing.T) {
	se := json.New(false)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	factory.FlushGaugesEvery(10)
	factory.RegisterGauge("gauge", nil)

	factory.GaugeSet("gauge", 10)
	factory.GaugeInc("gauge")
	factory.GaugeInc("gauge")
	factory.GaugeInc("gauge")
	factory.GaugeDec("gauge")
	factory.GaugeDec("gauge")
	factory.GaugeDec("gauge")
	factory.GaugeDec("gauge")
	factory.GaugeSet("gauge", -10)

	factory.Shutdown()

	if sh.tracer.Len() == 0 {
		t.Logf("Gauge did not get flushed")
		t.Fail()
	}
	for _, output := range sh.tracer.Show() {
		t.Logf("%s", output)
	}
}

func TestMultipleGauges(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	factory.FlushGaugesEvery(10)
	gaugesToTest := []string{"one", "two", "three"}
	gaugeTags := map[string]string{
		"gaugeTag": "one",
	}
	for _, name := range gaugesToTest {
		factory.RegisterGauge(name, gaugeTags)
	}

	for _, name := range gaugesToTest {
		factory.GaugeSet(name, 10)
		factory.GaugeInc(name)
		factory.GaugeInc(name)
		factory.GaugeInc(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeSet(name, -10)
	}

	factory.Shutdown()

	if sh.tracer.Len() == 0 {
		t.Logf("Gauge did not get flushed")
		t.Fail()
	}
	for _, output := range sh.tracer.Show() {
		t.Logf("%s", output)
	}
}

// This test we are really looking for a panic. something like calling on a nil value
// from a map.
func TestRemovingGauges(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	gaugesToTest := []string{"one", "two", "three"}
	gaugeTags := map[string]string{
		"gaugeTag": "one",
	}
	for _, name := range gaugesToTest {
		factory.RegisterGauge(name, gaugeTags)
	}

	for _, name := range gaugesToTest {
		factory.GaugeSet(name, 10)
		factory.GaugeInc(name)
		factory.GaugeInc(name)
		factory.GaugeInc(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeDec(name)
		factory.GaugeSet(name, -10)
	}

	for _, name := range gaugesToTest {
		factory.RemoveGauge(name)
	}

	for _, name := range gaugesToTest {
		factory.GaugeSet(name, 10)
		factory.GaugeDec(name)
		factory.GaugeSet(name, -10)
	}

	factory.Shutdown()

	if sh.tracer.Len() != 0 {
		t.Logf("Gauges did not flush")
		t.Fail()
	}
	for _, output := range sh.tracer.Show() {
		t.Logf("%s", output)
	}
}

func BenchmarkGauges(b *testing.B) {
	se := json.New(false)
	sh := devnull.New()
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { b.Logf("Factory Failed!"); b.FailNow() })

	gaugesToTest := []string{"one"}
	gaugeTags := map[string]string{
		"gaugeTag": "one",
	}
	for i := 0; i < b.N; i++ {
		for _, name := range gaugesToTest {
			factory.RegisterGauge(name, gaugeTags)
		}

		for _, name := range gaugesToTest {
			factory.GaugeSet(name, 10)
			factory.GaugeInc(name)
			factory.GaugeInc(name)
			factory.GaugeInc(name)
			factory.GaugeDec(name)
			factory.GaugeDec(name)
			factory.GaugeDec(name)
			factory.GaugeDec(name)
			factory.GaugeSet(name, -10)
		}

		for _, name := range gaugesToTest {
			factory.RemoveGauge(name)
		}
	}
	factory.Shutdown()
}

func TestNilGauge(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	factory.FlushGaugesEvery(2)
	time.Sleep(time.Millisecond * 3)
	factory.RemoveGauge("test")
	factory.Shutdown()

	if sh.tracer.Len() != 0 {
		t.Logf("Rouge metric in the buffer.\n%s", sh.tracer.Show())
	}
}

func TestTimers(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	start := time.Now()
	time.Sleep(time.Millisecond * 5)
	factory.Timer("test_timer_ns", TimerNS, time.Since(start), nil)
	factory.Timer("test_timer_µs", TimerUS, time.Since(start), nil)
	factory.Timer("test_timer_ms", TimerMS, time.Since(start), nil)
	factory.Timer("test_timer_sec", TimerSec, time.Since(start), nil)
	factory.Timer("test_timer_min", TimerMin, time.Since(start), nil)
	factory.Timer("test_timer_hour", TimerH, time.Since(start), nil)
	factory.Shutdown()

	t.Logf("%s", sh.tracer.Show())
}

type badSerializer struct{}

func (badSerializer) Counter(m *measurements.Counter) ([]byte, error) {
	return nil, fmt.Errorf("here is an error")
}
func (badSerializer) Timer(m *measurements.Timer) ([]byte, error) {
	return nil, fmt.Errorf("here is an error")
}
func (badSerializer) Gauge(m *measurements.Gauge) ([]byte, error) {
	return nil, fmt.Errorf("here is an error")
}
func (badSerializer) Poly(m *measurements.Poly) ([]byte, error) {
	return nil, fmt.Errorf("here is an error")
}

func TestFactoryError(t *testing.T) {
	se := &badSerializer{}
	sh := &tracing{tracer: gotracer.New()}
	gotError := false
	errFunc := func(error) {
		t.Logf("Caught factory error")
		gotError = true
	}
	factory := NewFactory("test", nil, se, sh, errFunc)
	factory.Counter("test", 1, 1, nil)
	factory.Shutdown()
	if !gotError {
		t.Logf("Factory got an error but didn't call error func")
		t.Fail()
	}
}

func TestPolyEmpty(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	p := factory.NewPoly("test_poly", nil, nil)
	factory.SubmitPoly(p)
	factory.Shutdown()

	if sh.tracer.Len() < 1 {
		t.Logf("Empty poly didn't make a measurement")
		t.Fail()
	}
	t.Logf("%s", sh.tracer.Show())
}

func TestPoly(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.Logf("Factory Failed!"); t.FailNow() })
	p := factory.NewPoly("test_poly", nil, nil)
	p.AddField("event", "start")
	p.AddField("event_message", "This is a app starting")
	p.AddTag("app_name", "potato")
	factory.SubmitPoly(p)
	factory.Shutdown()

	if sh.tracer.Len() < 1 {
		t.Logf("Poly didn't make a measurement")
		t.Fail()
	}
	t.Logf("%s", sh.tracer.Show())
}

func TestPolyStatsD(t *testing.T) {
	se := statsd.New(statsd.TaggingDataDog)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}

	factory := NewFactory("test", defaultTags, se, sh, func(err error) { t.Logf("Factory Failed! Error: %s", err); t.FailNow() })
	p := factory.NewPoly("poly_event", nil, nil)
	p.AddTag("event", "start")
	p.AddTag("event_message", "This is a app starting")
	p.AddTag("app_name", "potato")
	p.AddTag(statsd.MagicTagMetricType, "counter")
	p.AddField("start_event", 1)
	factory.SubmitPoly(p)
	factory.Shutdown()

	if sh.tracer.Len() < 1 {
		t.Logf("Poly didn't make a measurement")
		t.Fail()
	}
	t.Logf("%s", sh.tracer.Show())
}
