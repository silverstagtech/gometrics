package shippers

import (
	"fmt"
	"time"
)

// RateLimitedErrors is a function that allows you to rate limit errors functions from being fired.
// In some cases where the same error is continiously triggered you can put unneeded stress on the system.
// For example if a buffer is full and you have triggered the error more than once you don't need to do it
// for every error.
func RateLimitedErrors(timeLimit time.Duration, rateTrigger int, suppresionMessage string, errFunc func(error)) func(error) {
	timeNow := time.Now()
	errCount := 0
	suppresionMessageSent := false

	reset := func() {
		timeNow = time.Now()
		errCount = 0
		suppresionMessageSent = false
	}

	return func(err error) {
		errCount++
		if errCount > rateTrigger {
			if time.Since(timeNow) < timeLimit {
				if !suppresionMessageSent {
					err := fmt.Errorf("%s %s", suppresionMessage, err)
					errFunc(err)
					suppresionMessageSent = true
				}
				return
			}
			reset()
		}

		errFunc(err)
	}
}
