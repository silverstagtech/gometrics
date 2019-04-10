package gometrics

import (
	"sync"
	"time"
)

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
