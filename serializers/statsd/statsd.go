package statsd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/silverstagtech/gometrics/measurements"
)

// Tagging defines how you would like the tags to be applied to the metric
type Tagging string

const (
	// TaggingDataDog format
	// Tags are at the end with a preceding #. Tags are colon pairs and comma seperated.
	// Sample rate is optional.
	// As a side note telegraf is capable of digesting both in the same metric. They look for datadog
	// using the preceding # after splitting on a |
	//
	// https://docs.datadoghq.com/developers/dogstatsd/datagram_shell/
	//
	// Example:
	// metric_name:value|type[|sample_rate]|#tag:value,tag:value
	TaggingDataDog = "datadog"

	// TaggingInflux format
	// Influx pushed the tags into the middle of the metrics and have a = key value pair and comma seperated.
	// Sampling is option here.
	//
	// https://www.influxdata.com/blog/getting-started-with-sending-statsd-metrics-to-telegraf-influxdb/
	//
	// Example:
	// metric_name[,tag=value,tag=value]:value|type[|sample_rate]
	TaggingInflux = "influx"
	// TaggingNone will just drop the tags
	TaggingNone = "none"

	// MagicTagSampleRate is used to serialize the sample rate on polys
	MagicTagSampleRate = "sample_rate"
	// MagicTagMetricType is used to serialize the metric type on polys
	MagicTagMetricType = "metric_type"
)

var (
	statsdMetricTypes = map[string]string{
		"gauge":     "g",
		"set":       "s",
		"counter":   "c",
		"timing":    "ms",
		"histogram": "h",
	}
)

// StatsD is a serializer that will format the measurement into strings with single measurements
// per line. It is able to format the tags into Influx or DataDogs formatting.
type StatsD struct {
	taggingFormat        Tagging
	gaugesPreviousValues map[string]int64
}

// New returns a pointer to a StatD serializer that is ready to take measurements
func New(taggingFormat Tagging) *StatsD {
	return &StatsD{
		taggingFormat:        taggingFormat,
		gaugesPreviousValues: make(map[string]int64),
	}
}

// Counter formats counter measurements
func (sd *StatsD) Counter(m *measurements.Counter) ([]byte, error) {
	name := buildName(m.Prefix, m.Name)
	tags := mergeTags(sd.taggingFormat, m.Tags)
	formattedMetric := format(
		sd.taggingFormat,
		name,
		statsdMetricTypes["counter"],
		m.Value,
		m.Rate,
		tags,
	)
	return []byte(formattedMetric), nil
}

// Timer formats timer measurements
// Statsd timers seem to only work with ms. However other platforms work in
// any value. Therefore if the user wants to make use of statsd then all
// timers need to be set to precision ms. If it is not it will drop you
// metrics. (roll eyes...)
func (sd *StatsD) Timer(m *measurements.Timer) ([]byte, error) {
	if m.Unit != "ms" {
		return nil, fmt.Errorf("Statsd only supports ms precision on metrics, dropping %s", buildName(m.Prefix, m.Name))
	}
	name := buildName(m.Prefix, m.Name)
	tags := mergeTags(sd.taggingFormat, m.Tags)
	formattedMetric := format(
		sd.taggingFormat,
		name,
		statsdMetricTypes["timing"],
		int64(m.Value),
		1,
		tags,
	)
	return []byte(formattedMetric), nil
}

// Gauge formats Gauge measurements
// Gauges are special due to a bug...
// If a gauge was positive and then becomes negative, you first need to set it to zero,
// then set the value to the negative number. The same the other way around.
// What this means for us is that we need to track the values of the gauges from the
// last value. (roll eyes...)
// This could mean that we need to send 2 statsd measurements for 1 gauge.
func (sd *StatsD) Gauge(m *measurements.Gauge) ([]byte, error) {
	isPositive := func(v int64) bool {
		if v > 0 {
			return true
		}
		return false
	}

	isNegative := func(v int64) bool {
		if v < 0 {
			return true
		}
		return false
	}

	name := buildName(m.Prefix, m.Name)
	tags := mergeTags(sd.taggingFormat, m.Tags)

	setZero := false
	previousValue, ok := sd.gaugesPreviousValues[name]
	if !ok {
		sd.gaugesPreviousValues[name] = m.Value
	} else {
		switch {
		case previousValue == 0:
		case isPositive(previousValue) && isNegative(m.Value):
			setZero = true
		case isNegative(previousValue) && isPositive(m.Value):
			setZero = true
		}
	}

	out := []string{}

	if setZero {
		zeroMetric := format(
			sd.taggingFormat,
			name,
			statsdMetricTypes["gauge"],
			0,
			1,
			tags,
		)
		out = append(out, zeroMetric)
	}

	formattedMetric := format(
		sd.taggingFormat,
		name,
		statsdMetricTypes["gauge"],
		m.Value,
		1,
		tags,
	)
	out = append(out, formattedMetric)

	return []byte(strings.Join(out, "\n")), nil
}

// Poly formats poly measurements
// StatsD doesn't really understand the idea of multiple measurements in a single
// line. Therefore if you want to use a Polymorphic measurement then you need to
// have only a single type in the measurement.
// Polymorphic measurements get broken down into multiple statsd lines each containing
// the values from the provided fields. They still have to be the same type...
// There are some exceptions, due to interface{} being ANYTHING in go, we only accept
// certian types and if its not one of them a error is returned a the message is dropped.
// Consider yourself warned...
// Magic Tags
// Because polymorphic measurements don't really translate well into statsd we need to
// add some hints into the tags to tell the serializer what to do with the measurements.
// I call these magic tags. They provide the statsd type and sample rate.
// Below is how they work. The heading is what the tag needs to be. Always lower case.
//
// metric_type
// Valid values: "gauge", "set", "counter", "timing", "ms", "histogram"
// Description: Is used to map out the type that is attached to the statsd metric.
// Required: true
//
// sample_rate
// Valid values: A float32 as a string.
// Description:  Used in place of the sample rate in the statsd measurement
// Required: false
func (sd *StatsD) Poly(m *measurements.Poly) ([]byte, error) {
	if ok := validPolyFields(m.Fields); !ok {
		return nil, fmt.Errorf("Invalid fields types used")
	}
	name := buildName(m.Prefix, m.Name)
	realTags, metricType, sampleRate, ok := digestMagicTags(m.Tags)
	if !ok {
		return nil, fmt.Errorf("magic tag was not a valid value")
	}
	tags := mergeTags(sd.taggingFormat, realTags)

	output := []string{}

	var sampleFloat float32 = 1
	if sampleRate != "" {
		rate, err := strconv.ParseFloat(sampleRate, 32)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse the sample rate as a float32. Error: %s", err)
		}
		sampleFloat = float32(rate)
	}

	for _, field := range m.Fields {
		formattedMetric := format(
			sd.taggingFormat,
			name,
			metricType,
			field,
			sampleFloat,
			tags,
		)
		output = append(output, formattedMetric)
	}
	return []byte(strings.Join(output, "\n")), nil
}

func validPolyFields(values map[string]interface{}) bool {
	for _, value := range values {
		switch value.(type) {
		case int:
		case uint:
		case int8:
		case uint8:
		case int16:
		case uint16:
		case int32:
		case uint32:
		case int64:
		case uint64:
		case float64:
		case float32:
		case string:
		default:
			return false
		}
	}
	return true
}

func digestMagicTags(tags map[string]string) (newTags map[string]string, metricType, sampleRate string, ok bool) {
	newTags = make(map[string]string)
	for key, value := range tags {
		switch key {
		case MagicTagMetricType:
			metricType, ok = statsdMetricTypes[value]
			if !ok {
				// found a metric_type tag but is not in the list of approved names.
				return
			}
		case MagicTagSampleRate:
			sampleRate = value
		default:
			newTags[key] = value
		}
	}
	return
}

func buildName(prefix, name string) string {
	return prefix + "_" + name
}

func mergeTags(formatting Tagging, tags map[string]string) string {
	if tags == nil {
		return ""
	}

	switch formatting {
	case TaggingDataDog:
		return datadogTags(tags)
	case TaggingInflux:
		return influxTags(tags)
	case TaggingNone:
		return ""
	default:
		return ""
	}
}

func datadogTags(tags map[string]string) string {
	return fmt.Sprintf("|#%s", pairTags(tags, ":"))
}

func influxTags(tags map[string]string) string {
	return fmt.Sprintf(",%s", pairTags(tags, "="))
}

func pairTags(t map[string]string, seperator string) string {
	returnTags := make([]string, len(t))
	index := 0
	for name, value := range t {
		returnTags[index] = fmt.Sprintf("%s%s%s", name, seperator, value)
		index++
	}
	return strings.Join(returnTags, ",")
}

// format is used to create the metric string. It will do one value at a time. So if you have many fields like in a poly you will need
// to call it many times.
// The name needs to be merged before as well as the tags being serialized. This is to stop this function from having to do it
// on every call with multiple values.
// Formatting:
// Influx tagging:
//   metric_name[,tag=value,tag=value]:value|type[|sample_rate]
// DataDog:
//   metric_name:value|type[|sample_rate]|#tag:value,tag:value
func format(tagging Tagging, name, statsdType string, value interface{}, sampleRate float32, tags string) string {
	nameValues := ""
	if tagging == TaggingInflux {
		nameValues = fmt.Sprintf("%s%s:%v|%s", name, tags, value, statsdType)
	} else {
		nameValues = fmt.Sprintf("%s:%v|%s", name, value, statsdType)
	}

	sampleRateString := ""
	if sampleRate > 0 && sampleRate < 1 {
		sampleRateString = fmt.Sprintf("|@%.2f", sampleRate)
	}

	if tagging == TaggingDataDog {
		return nameValues + sampleRateString + tags
	}
	return nameValues + sampleRateString
}
