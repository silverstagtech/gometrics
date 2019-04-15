package json

import (
	"encoding/json"

	"github.com/silverstagtech/gometrics/measurements"
)

func buildName(prefix, name string) string {
	return prefix + "_" + name
}

type marshaller func(interface{}) ([]byte, error)

type JSONSerializer struct {
	marshal func(interface{}) ([]byte, error)
}

func New(prettyPrint bool) *JSONSerializer {
	var f marshaller
	if prettyPrint {
		f = prettyprinter
	} else {
		f = json.Marshal
	}
	return &JSONSerializer{
		marshal: f,
	}
}

func prettyprinter(in interface{}) ([]byte, error) {
	return json.MarshalIndent(in, "", "    ")
}

type counter struct {
	Type  string            `json:"metric_type"`
	Name  string            `json:"name"`
	Tags  map[string]string `json:"tags"`
	Rate  float32           `json:"rate"`
	Value int64             `json:"value"`
}

func (js *JSONSerializer) Counter(m *measurements.Counter) ([]byte, error) {
	return js.marshal(
		&counter{
			Type:  "counter",
			Name:  buildName(m.Prefix, m.Name),
			Tags:  m.Tags,
			Rate:  m.Rate,
			Value: m.Value,
		},
	)
}

type timer struct {
	Type  string            `json:"metric_type"`
	Name  string            `json:"name"`
	Tags  map[string]string `json:"tags"`
	Value int64             `json:"value"`
	Unit  string            `json:"units"`
}

func (js *JSONSerializer) Timer(m *measurements.Timer) ([]byte, error) {
	return js.marshal(
		&timer{
			Type:  "timer",
			Name:  buildName(m.Prefix, m.Name),
			Tags:  m.Tags,
			Value: int64(m.Value),
			Unit:  m.Unit,
		},
	)
}

type gauge struct {
	Type  string            `json:"metric_type"`
	Name  string            `json:"name"`
	Tags  map[string]string `json:"tags"`
	Value int64             `json:"value"`
}

func (js *JSONSerializer) Gauge(m *measurements.Gauge) ([]byte, error) {
	return js.marshal(
		&gauge{
			Type:  "gauge",
			Name:  buildName(m.Prefix, m.Name),
			Tags:  m.Tags,
			Value: m.Value,
		},
	)
}

type poly struct {
	Type   string                 `json:"metric_type"`
	Name   string                 `json:"name"`
	Tags   map[string]string      `json:"tags"`
	Fields map[string]interface{} `json:"fields"`
}

func (js *JSONSerializer) Poly(m *measurements.Poly) ([]byte, error) {
	return js.marshal(
		&poly{
			Type:   "poly",
			Name:   buildName(m.Prefix, m.Name),
			Tags:   m.Tags,
			Fields: m.Fields,
		},
	)
}
