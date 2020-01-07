package gometrics

import (
	"math/rand"
)

// mergeTags needs to return a new map everytime because go passes maps by reference
// and they get screwed up as we add new tags to it everytime this function is called.
func mergeTags(defaultTags, MetricTags map[string]string) map[string]string {
	output := make(map[string]string)

	if defaultTags != nil {
		for key, value := range defaultTags {
			output[key] = value
		}
	}

	if MetricTags != nil {
		for key, value := range MetricTags {
			output[key] = value
		}
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
