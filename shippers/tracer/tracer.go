package tracer

import "github.com/silverstagtech/gotracer"

type Tracer struct {
	tracing *gotracer.Tracer
}

func New() *Tracer {
	return &Tracer{
		tracing: gotracer.New(),
	}
}

func (tr *Tracer) Ship(b []byte) {
	tr.tracing.SendBytes(b)
}

func (tr *Tracer) Shutdown() chan struct{} {
	c := make(chan struct{}, 1)
	close(c)
	return c
}

func (tr *Tracer) ExposeTracing() *gotracer.Tracer {
	return tr.tracing
}
