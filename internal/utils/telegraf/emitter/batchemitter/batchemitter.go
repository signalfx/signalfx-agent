package batchemitter

import (
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/telegraf/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/utils/telegraf/measurement"
)

// BatchEmitter gathers a batch of telegraf measurements that can be modified
// and then emitted
type BatchEmitter struct {
	*baseemitter.BaseEmitter
	lock         *sync.Mutex
	Measurements []*measurement.Measurement
}

// Add parses measurements from telegraf and saves them to the struct member
// Measurements
func (B *BatchEmitter) Add(m string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType,
	originalMetricType string, t ...time.Time) {
	B.lock.Lock()
	B.Measurements = append(B.Measurements, &measurement.Measurement{
		Measurement:        m,
		Fields:             fields,
		Tags:               utils.CloneAndFilterStringMapWithFunc(tags, B.BaseEmitter.FilterTags),
		MetricType:         metricType,
		OriginalMetricType: originalMetricType,
		T:                  t,
	})
	B.lock.Unlock()
}

// Send the metrics in the batch through the agent
func (B *BatchEmitter) Send() {
	B.lock.Lock()
	for _, m := range B.Measurements {
		B.BaseEmitter.Add(m.Measurement, m.Fields, m.Tags, m.MetricType,
			m.OriginalMetricType, m.T...)
	}
	B.lock.Unlock()
}

// NewEmitter returns a new BatchEmitter that gathers a batch of telegraf
// measurements so they can be modified and then emitted
func NewEmitter() *BatchEmitter {
	return &BatchEmitter{
		BaseEmitter:  baseemitter.NewEmitter(),
		lock:         &sync.Mutex{},
		Measurements: []*measurement.Measurement{},
	}
}
