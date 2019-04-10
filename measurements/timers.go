package measurements

import "time"

type Timer struct {
	Prefix   string
	Name     string
	Duration time.Duration
	Tags     map[string]string
}
