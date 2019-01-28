package measurement

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Measurement is a storage struct for storing measurements from telegraf
// plugins
type Measurement struct {
	Measurement        string
	Fields             map[string]interface{}
	Tags               map[string]string
	MetricType         datapoint.MetricType
	OriginalMetricType string
	Timestamps         []time.Time
}

// RenameFieldWithTag - takes the value of a specified tag and uses it to rename a specified field
// the tag is deleted and the original field name is overwritten
func (m *Measurement) RenameFieldWithTag(tagName string, fieldName string, replacer *strings.Replacer) {
	if tagVal, ok := m.Tags[tagName]; ok {
		tagVal = replacer.Replace(tagVal)
		if val, ok := m.Fields[fieldName]; ok && tagVal != "" {
			m.Fields[tagVal] = val
			delete(m.Fields, fieldName)
			delete(m.Tags, tagName)
		}
	}
}

// New creates a new measurement from telegraf measurement name, fields, tags, etc.
func New(measurement string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType,
	originalMetricType string, t ...time.Time) *Measurement {
	return &Measurement{
		Measurement: measurement,
		Fields:      fields,
		// telegraf doc says that tags are owned by the calling plugin and they
		// shouldn't be mutated.  So we copy the tags map
		Tags:               utils.CloneStringMap(tags),
		MetricType:         metricType,
		OriginalMetricType: originalMetricType,
		Timestamps:         t,
	}
}
