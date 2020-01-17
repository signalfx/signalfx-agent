package memory

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
	logger logrus.FieldLogger
}

// EmitDatapoints emits a set of memory datapoints
func (m *Monitor) emitDatapoints() {
	// mem.VirtualMemory is a gopsutil function
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.logger.WithError(err).Errorf("Unable to collect memory stats")
		return
	}

	// all platforms
	dps := []*datapoint.Datapoint{
		datapoint.New("memory.utilization", nil, datapoint.NewFloatValue(memInfo.UsedPercent), datapoint.Gauge, time.Time{}),
		datapoint.New("memory.used", nil, datapoint.NewIntValue(int64(memInfo.Used)), datapoint.Gauge, time.Time{}),
	}

	dps = append(dps, m.makeMemoryDatapoints(memInfo, nil)...)

	m.Output.SendDatapoints(dps...)
}

// Configure is the main function of the monitor, it will report host metadata
// on a varied interval
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(log.Fields{"monitorType": monitorType})

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		m.emitDatapoints()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
