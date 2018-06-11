package measurement

import (
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
	T                  []time.Time
}
