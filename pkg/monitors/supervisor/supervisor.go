package supervisor

import (
	"context"
	"time"

	"github.com/mattn/go-xmlrpc"
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`
	// The Supervisor XML-RPC API URL (i.e. `http://localhost:9001/RPC2`).
	URL string `yaml:"url" validate:"required"`
}

// Monitor that collect metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel func()
	logger logrus.FieldLogger
}

// Process contains Supervisor properties
type Process struct {
	Name  string `xmlrpc:"name"`
	Group string `xmlrpc:"group"`
	State int    `xmlrpc:"state"`
}

// Configure and kick off internal metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.logger = logrus.WithFields(logrus.Fields{"monitorType": monitorType})

	// Start the metric gathering process here
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	utils.RunOnInterval(ctx, func() {

		client := xmlrpc.NewClient(conf.URL)
		res, err := client.Call("supervisor.getAllProcessInfo")
		if err != nil {
			m.logger.WithError(err).Error("unable to call supervisor xmlrpc")
			return
		}

		var process Process
		for _, p := range res.(xmlrpc.Array) {
			for k, v := range p.(xmlrpc.Struct) {
				switch k {
				case "name":
					process.Name = v.(string)
				case "group":
					process.Group = v.(string)
				case "state":
					process.State = v.(int)
				}
			}
			dimensions := map[string]string{
				"name":  process.Name,
				"group": process.Group,
			}
			m.Output.SendDatapoints([]*datapoint.Datapoint{
				datapoint.New(
					supervisorState,
					dimensions,
					datapoint.NewIntValue(int64(process.State)),
					datapoint.Gauge,
					time.Time{},
				),
			}...)
		}
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
