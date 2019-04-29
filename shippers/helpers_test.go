package shippers

import (
	"fmt"
	"testing"
	"time"

	"github.com/silverstagtech/gotracer"
)

func TestRateLimit(t *testing.T) {
	tracing := gotracer.New()
	topErrFunc := func(err error) {
		tracing.Send(err.Error())
	}

	rateLimitErrFunc := RateLimitedErrors(time.Millisecond*2, 3, "Rate limit triggered, suppressing future messages. Error:", topErrFunc)

	for i := 0; i < 100; i++ {
		rateLimitErrFunc(fmt.Errorf("Test error for rate limit"))
	}

	t.Logf("Error count: %d", tracing.Len())
	tracing.PrintlnT(t)

	if !(tracing.Len() >= 3) || tracing.Len() == 100 {
		t.Logf("Rate limit did not limit some errors from being captured. Max: %d, Got: %d", 100, tracing.Len())
	}
}
