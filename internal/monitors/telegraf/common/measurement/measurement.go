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
func (m *Measurement) RenameFieldWithTag(tagName string, fieldName string, replacer *strings.Replacer) {
	var tagVal string
	var ok bool
	if tagVal, ok = m.Tags[tagName]; ok {
		tagVal = replacer.Replace(strings.ToLower(tagVal))
		delete(m.Tags, tagName)
	}
	if val, ok := m.Fields[fieldName]; ok && tagVal != "" {
		delete(m.Fields, fieldName)
		m.Fields[tagVal] = val
	}
}
