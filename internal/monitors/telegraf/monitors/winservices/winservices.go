package winservices

import (
	"context"
	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/win_services"

// MONITOR(telegraf/win_services): This monitor reports metrics about Windows services.
// This monitor is based on the Telegraf win_services plugin.  More information about the Telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/win_services).
//
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: telegraf/win_services  # monitor all services
// ```
//
// ```yaml
// monitors:
//  - type: telegraf/win_services
//    serviceNames:
//      - exampleService1  # only monitor exampleService1
// ```
//

// Metrics

// GAUGE(win_services.state): The current state of the service: 1 `stopped`, 2 `start pending`, 3 `stop pending`, 4 `running`, 5 `continue pending`, 6 `pause pending`, or 7 `paused`

// GAUGE(win_services.startup_mode): The startup mode configured for the service: 0 `boot start`, 1 `system start`, 2 `auto start`, 3 `demand start`, or 4 `disabled`

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
	// Names of services to monitor.  All services will be monitored if none are specified.
	ServiceNames []string `yaml:"serviceNames"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel context.CancelFunc
}

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["win_services"]

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
