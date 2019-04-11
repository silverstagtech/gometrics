package statsd

import (
	"strings"
	"testing"

	"github.com/silverstagtech/gometrics/measurements"
	"github.com/silverstagtech/gotracer"
)

func TestStatsDCounter(t *testing.T) {
	tags := map[string]string{
		"app": "amazing_app",
	}
	m := &measurements.Counter{
		Prefix: "testing",
		Name:   "measurement",
		Value:  1,
		Rate:   0.25,
		Tags:   tags,
	}
	testers := map[Tagging]string{
		TaggingDataDog: "testing_measurement:1|c|@0.25|#app:amazing_app",
		TaggingInflux:  "testing_measurement,app=amazing_app:1|c|@0.25",
		TaggingNone:    "testing_measurement:1|c|@0.25",
	}

	sd := &StatsD{}

	for taggingPolicy, matcher := range testers {
		sd.taggingFormat = taggingPolicy
		b, _ := sd.Counter(m)
		t.Logf("%s: %s", taggingPolicy, b)
		if !strings.Contains(string(b), matcher) {
			t.Logf("Formatting in %s did not give the correct result.\nGot: %s\nWant: %s", taggingPolicy, string(b), matcher)
			t.Fail()
		}
	}
}

func BenchmarkCounterDD(b *testing.B) {
	tags := map[string]string{
		"app": "amazing_app",
	}
	m := &measurements.Counter{
		Prefix: "testing",
		Name:   "measurement",
		Value:  1,
		Rate:   0.25,
		Tags:   tags,
	}

	sd := &StatsD{
		taggingFormat: TaggingDataDog,
	}

	for i := 0; i < b.N; i++ {
		_, _ = sd.Counter(m)
	}
}

func BenchmarkCounterIf(b *testing.B) {
	tags := map[string]string{
		"app": "amazing_app",
	}
	m := &measurements.Counter{
		Prefix: "testing",
		Name:   "measurement",
		Value:  1,
		Rate:   0.25,
		Tags:   tags,
	}

	sd := &StatsD{
		taggingFormat: TaggingInflux,
	}

	for i := 0; i < b.N; i++ {
		_, _ = sd.Counter(m)
	}
}

func TestStatsTimer(t *testing.T) {
	tags := map[string]string{
		"app": "amazing_app",
	}

	m := &measurements.Timer{
		Prefix: "testing",
		Name:   "measurement",
		Value:  1550,
		Unit:   "ms",
		Tags:   tags,
	}
	testers := map[Tagging]string{
		TaggingDataDog: "testing_measurement:1550|ms|#app:amazing_app",
		TaggingInflux:  "testing_measurement,app=amazing_app:1550|ms",
		TaggingNone:    "testing_measurement:1550|ms",
	}

	sd := &StatsD{}

	for taggingPolicy, matcher := range testers {

		sd.taggingFormat = taggingPolicy
		b, _ := sd.Timer(m)
		t.Logf("%s: %s", taggingPolicy, b)
		if !strings.Contains(string(b), matcher) {
			t.Logf("Formatting in %s did not give the correct result.\nGot: %s\nWant: %s", taggingPolicy, string(b), matcher)
			t.Fail()
		}
	}
}

func TestStatsGauges(t *testing.T) {
	tags := map[string]string{
		"app":    "amazing_app",
		"region": "europe",
	}
	m := &measurements.Gauge{
		Prefix: "testing",
		Name:   "measurement",
		Value:  1,
		Tags:   tags,
	}

	sd := New(TaggingDataDog)
	tracer := gotracer.New()
	b, _ := sd.Gauge(m)
	tracer.Send(string(b))

	m.Value = -1
	b, _ = sd.Gauge(m)
	for _, s := range strings.Split(string(b), "\n") {
		tracer.Send(s)
	}

	t.Logf("%s", strings.Join(tracer.Show(), "\n"))

	if tracer.Len() != 3 {
		t.Logf("Did not reset the gauge to zero before going negative. Got: %d, Want: %d", tracer.Len(), 3)
		t.Fail()
	}
}

func TestStatsPoly(t *testing.T) {
	tags := map[string]string{
		"app":              "amazing_app",
		"region":           "europe",
		MagicTagMetricType: "counter",
		MagicTagSampleRate: "0.26",
	}
	fields := map[string]interface{}{
		"f1": 1,
		"f2": "two",
		"f3": 3.5,
		"f4": int64(123456789),
	}
	m := &measurements.Poly{
		Prefix: "testing",
		Name:   "measurement",
		Fields: fields,
		Tags:   tags,
	}

	sd := New(TaggingDataDog)
	tracer := gotracer.New()
	b, err := sd.Poly(m)
	if err != nil {
		t.Logf("error: %s", err)
		t.Fail()
	}
	for _, s := range strings.Split(string(b), "\n") {
		tracer.Send(s)
	}

	t.Logf("%s", strings.Join(tracer.Show(), "\n"))

	if tracer.Len() != 4 {
		t.Logf("Did not get the correct number of measurements from passed in poly. Got: %d, Want: %d", tracer.Len(), 4)
		t.Fail()
	}
}

func BenchmarkStatsPoly(b *testing.B) {
	tags := map[string]string{
		"app":              "amazing_app",
		"region":           "europe",
		MagicTagMetricType: "counter",
		MagicTagSampleRate: "0.26",
	}
	fields := map[string]interface{}{
		"f1": 1,
		"f2": "two",
		"f3": 3.5,
		"f4": int64(123456789),
	}
	m := &measurements.Poly{
		Prefix: "testing",
		Name:   "measurement",
		Fields: fields,
		Tags:   tags,
	}

	sd := New(TaggingDataDog)
	for i := 0; i < b.N; i++ {
		_, err := sd.Poly(m)
		if err != nil {
			b.Logf("error: %s", err)
			b.Fail()
		}
	}
}

func TestBadPoly(t *testing.T) {
	tags := map[string]string{
		"app":              "amazing_app",
		"region":           "europe",
		MagicTagMetricType: "counter",
		MagicTagSampleRate: "0.26",
	}
	fields := map[string]interface{}{
		"f1": map[string]interface{}{
			"bad": "field",
		},
	}
	m := &measurements.Poly{
		Prefix: "testing",
		Name:   "measurement",
		Fields: fields,
		Tags:   tags,
	}

	sd := New(TaggingDataDog)
	_, err := sd.Poly(m)
	if err == nil {
		t.Logf("Passing in a unsupported field type did not give a error.")
		t.Fail()
	}
}
