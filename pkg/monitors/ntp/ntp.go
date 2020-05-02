package ntp

import (
	"context"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/beevik/ntp"
)

const minIntervalSeconds = 1800

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`
	// The host/ip address of the NTP server.
	Host string `yaml:"host" default:"pool.ntp.org"`
	// The port of the NTP server.
	Port int `yaml:"port" default:"123"`
	// NTP protocol version to.
	Version int `yaml:"version" default:"4"`
	// Timeout in seconds for the request.
	Timeout uint16 `yaml:"timeout" default:"5"`
}

// Monitor that collect metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel func()
	logger logrus.FieldLogger
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": monitorType})
	// respect terms of service https://www.pool.ntp.org/tos.html
	if conf.IntervalSeconds < minIntervalSeconds {
		m.logger.WithField("IntervalSeconds", minIntervalSeconds).Info("overrides to minimum interval")
		conf.IntervalSeconds = minIntervalSeconds
	}
	// Start the metric gathering process here
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {
		options := ntp.QueryOptions{Version: conf.Version, Port: conf.Port, Timeout: time.Duration(conf.Timeout) * time.Second}
		response, err := ntp.QueryWithOptions(conf.Host, options)
		if err != nil {
			m.logger.WithError(err).Error("unable to get ntp statistics")
			return
		}
		clockOffset := response.ClockOffset.Seconds()
		m.Output.SendDatapoints([]*datapoint.Datapoint{
			datapoint.New("ntp.offset", map[string]string{"ntp": conf.Host}, datapoint.NewFloatValue(clockOffset), datapoint.Gauge, time.Time{}),
		}...)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown the monitor
func (m *Monitor) Shutdown() {
	// Stop any long-running go routines here
	if m.cancel != nil {
		m.cancel()
	}
}
