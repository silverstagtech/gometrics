package gometrics

import (
	"math/rand"
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
