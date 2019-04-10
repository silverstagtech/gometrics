package gometrics

import (
	"testing"

	"github.com/silverstagtech/gotracer"

	"github.com/silverstagtech/gometrics/serializers/json"
	"github.com/silverstagtech/gometrics/shippers/stdout"
)

type tracing struct {
	tracer *gotracer.Tracer
}

func (tr tracing) Ship(b []byte) {
	tr.tracer.Send(string(b))
}

func TestCounter(t *testing.T) {
	se := json.New(true)
	sh := &tracing{tracer: gotracer.New()}
	defaultTags := map[string]string{
		"testone": "1",
	}
	factory := NewFactory("test", defaultTags, se, sh, func(error) { t.FailNow() })

	factory.Counter("counter", 100, 1, nil)

	t.Logf("%v", sh.tracer.Show())
}

func BenchmarkCounter(b *testing.B) {
	se := json.New(false)
	//sh := devnull.New()
	sh := stdout.New()
	defaultTags := map[string]string{
		"testone": "1",
	}
	factory := NewFactory("test", defaultTags, se, sh, func(error) { b.FailNow() })
	for n := 0; n < b.N; n++ {
		factory.Counter("counter", 100, 1, nil)
	}
}
