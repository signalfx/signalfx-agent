package measurement

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
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
func RenameFieldWithTag(m *Measurement, tagName string, fieldName string, replacer *strings.Replacer) {
	if tagVal, ok := m.Tags[tagName]; ok {
		tagVal = replacer.Replace(strings.ToLower(tagVal))
		if val, ok := m.Fields[fieldName]; ok && tagVal != "" {
			m.Fields[tagVal] = val
			delete(m.Fields, fieldName)
			delete(m.Tags, tagName)
		}
	}
}
