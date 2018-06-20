package batchemitter

import (
	"sync"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
	"github.com/signalfx/signalfx-agent/internal/utils"
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
		Timestamps:         t,
	})
	b.lock.Unlock()
}

// Send the metrics in the batch through the agent
func (b *BatchEmitter) Send() {
	b.lock.Lock()
	for _, m := range b.Measurements {
		b.BaseEmitter.Add(m.Measurement, m.Fields, m.Tags, m.MetricType,
			m.OriginalMetricType, m.Timestamps...)
	}
	b.lock.Unlock()
}

// NewEmitter returns a new BatchEmitter that gathers a batch of telegraf
// measurements so they can be modified and then emitted
func NewEmitter(Output types.Output, Logger log.FieldLogger) *BatchEmitter {
	return &BatchEmitter{
		BaseEmitter:  baseemitter.NewEmitter(Output, Logger),
		Measurements: []*measurement.Measurement{},
	}
}
