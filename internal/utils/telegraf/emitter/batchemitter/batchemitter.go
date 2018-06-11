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
	lock         sync.Mutex
	Measurements []*measurement.Measurement
}

// Add parses measurements from telegraf and saves them to the struct member
// Measurements
func (b *BatchEmitter) Add(m string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType,
	originalMetricType string, t ...time.Time) {
	b.lock.Lock()
	b.Measurements = append(b.Measurements, &measurement.Measurement{
		Measurement:        m,
		Fields:             fields,
		Tags:               utils.CloneAndFilterStringMapWithFunc(tags, b.BaseEmitter.FilterTags),
		MetricType:         metricType,
		OriginalMetricType: originalMetricType,
		T:                  t,
	})
	b.lock.Unlock()
}

// Send the metrics in the batch through the agent
func (b *BatchEmitter) Send() {
	b.lock.Lock()
	for _, m := range b.Measurements {
		b.BaseEmitter.Add(m.Measurement, m.Fields, m.Tags, m.MetricType,
			m.OriginalMetricType, m.T...)
	}
	b.lock.Unlock()
}

// NewEmitter returns a new BatchEmitter that gathers a batch of telegraf
// measurements so they can be modified and then emitted
func NewEmitter() *BatchEmitter {
	return &BatchEmitter{
		BaseEmitter:  baseemitter.NewEmitter(),
		lock:         sync.Mutex{},
		Measurements: []*measurement.Measurement{},
	}
}
