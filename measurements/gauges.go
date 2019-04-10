package measurements

type Gauge struct {
	Prefix string
	Name   string
	Value  int64
	Tags   map[string]string
}
