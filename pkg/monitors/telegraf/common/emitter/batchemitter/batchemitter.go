package batchemitter

import (
	"sync"

	"github.com/influxdata/telegraf"
	"github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	log "github.com/sirupsen/logrus"
)

// BatchEmitter gathers a batch of telegraf measurements that can be modified
// and then emitted
type BatchEmitter struct {
	*baseemitter.BaseEmitter
	lock    sync.Mutex
	Metrics []telegraf.Metric
}

// AddMetric takes a telegraf metric and saves it to the struct member
// Metrics
func (b *BatchEmitter) AddMetric(m telegraf.Metric) {
	b.lock.Lock()
	b.Metrics = append(b.Metrics, m)
	b.lock.Unlock()
}

// Send the metrics in the batch through the agent
func (b *BatchEmitter) Send() {
	b.lock.Lock()
	for _, m := range b.Metrics {
		b.BaseEmitter.AddMetric(m)
	}
	b.Metrics = b.Metrics[:0]
	b.lock.Unlock()
}

// NewEmitter returns a new BatchEmitter that gathers a batch of telegraf
// measurements so they can be modified and then emitted
func NewEmitter(output types.Output, logger log.FieldLogger) *BatchEmitter {
	return &BatchEmitter{
		BaseEmitter: baseemitter.NewEmitter(output, logger),
		Metrics:     []telegraf.Metric{},
	}
}
