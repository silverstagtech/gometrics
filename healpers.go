package gometrics

import (
	"math/rand"
	"sync"
	"time"
)

func mergeTags(defaultTags, MetricTags map[string]string) map[string]string {
	if defaultTags == nil {
		return MetricTags
	}

	if MetricTags == nil {
		return defaultTags
	}

	output := defaultTags
	for key, value := range MetricTags {
		output[key] = value
	}

	return output
}

func gatekeeper(rate float32) bool {
	if rate >= 1 {
		return true
	}
	if rand.Float32() > rate {
		return true
	}
	return false
}

type simpleTicker struct {
	sleeper time.Duration
	stopped bool
	c       chan struct{}
	sync.RWMutex
}

func newSimpleTicker(milliseconds int) *simpleTicker {
	return &simpleTicker{
		sleeper: time.Millisecond * time.Duration(milliseconds),
	}
}

func (st *simpleTicker) start() {
	stopped := func() bool {
		st.RLock()
		defer st.RUnlock()
		return st.stopped
	}

	st.Lock()
	st.c = make(chan struct{}, 1)
	st.stopped = false
	st.Unlock()

	go func() {
		for {
			if stopped() {
				close(st.c)
				return
			}
			time.Sleep(st.sleeper)
			select {
			case st.c <- struct{}{}:
			default:
				continue
			}
		}
	}()
}

func (st *simpleTicker) stop() {
	st.Lock()
	defer st.Unlock()
	st.stopped = true
}
