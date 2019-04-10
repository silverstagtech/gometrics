package measurements

// Poly or polymorphic measurements have many data points on them.
// They are useful when you want to measure some thing a bit more complicated than
// a single data point.
// ie. Memory has "used" and "free" which each be a field and sent in a single
// measurement.
// These can be populated as you get data and sent once completely populated.
type Poly struct {
	Prefix string
	Name   string
	Fields map[string]interface{}
	Tags   map[string]string
}

// NewPoly is returns a pointer to a Polymorphic measurement.
// NewPoly gives you a starter measurement and thus fields and tags can be nil
// and added later with AddTag(s) and AddField(s).
func NewPoly(prefix, name string, fields map[string]interface{}, tags map[string]string) *Poly {
	p := &Poly{
		Prefix: prefix,
		Name:   name,
		Fields: make(map[string]interface{}),
		Tags:   make(map[string]string),
	}
	p.AddTags(tags)
	p.AddFields(fields)
	return p
}

// AddTags allows you to add multiple tags to the measurement
func (p *Poly) AddTags(tags map[string]string) {
	if tags == nil {
		return
	}

	for key, value := range tags {
		p.Tags[key] = value
	}
}

// AddTag allows you to add a single tag to the measurement
func (p *Poly) AddTag(key, value string) {
	p.Tags[key] = value
}

// AddFields allows you add multiple metrics to the measurement
func (p *Poly) AddFields(fields map[string]interface{}) {
	if fields == nil {
		return
	}

	for key, value := range fields {
		p.Fields[key] = value
	}
}

// AddField allows you to add a single field to the measurement
func (p *Poly) AddField(key string, value interface{}) {
	p.Fields[key] = value
}
