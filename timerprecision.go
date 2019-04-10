package gometrics

import "time"

// TimerPrecision is used to shift the timers down to a desired precision
type TimerPrecision string

var (
	// TimerNS is Nanoseconds
	TimerNS TimerPrecision = "ns"
	// TimerUS is Microseconds
	TimerUS TimerPrecision = "μs"
	// TimerMS is Milliseconds
	TimerMS TimerPrecision = "ms"
	// TimerSec is Seconds
	TimerSec TimerPrecision = "s"
	// TimerMin is Minute
	TimerMin TimerPrecision = "m"
	// TimerH is Minute
	TimerH TimerPrecision = "h"
)

func timerUnit(unit TimerPrecision) (time.Duration, string) {
	switch unit {
	case TimerNS:
		return time.Nanosecond, string(TimerNS)
	case TimerUS:
		return time.Microsecond, string(TimerUS)
	case TimerMS:
		return time.Millisecond, string(TimerMS)
	case TimerSec:
		return time.Second, string(TimerSec)
	case TimerMin:
		return time.Minute, string(TimerMin)
	case TimerH:
		return time.Hour, string(TimerH)
	default:
		return time.Nanosecond, string(TimerNS)
	}
}
