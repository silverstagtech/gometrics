package gometrics

import (
	"testing"
	"time"

	"github.com/silverstagtech/gotracer"

	"github.com/silverstagtech/gometrics/serializers/json"
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
