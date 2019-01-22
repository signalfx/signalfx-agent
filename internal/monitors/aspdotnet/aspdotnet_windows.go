// +build windows

package aspdotnet

import (
	"context"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/batchemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters/perfhelper"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var metricMap = map[string]*perfhelper.MetricMapper{
	"asp_net.application_restarts":                                  {Name: "asp_net.application_restarts", Type: datapoint.Counter},
	"asp_net.applications_running":                                  {Name: "asp_net.applications_running"},
	"asp_net.requests_current":                                      {Name: "asp_net.requests_current"},
	"asp_net.requests_queued":                                       {Name: "asp_net.requests_queued"},
	"asp_net.requests_rejected":                                     {Name: "asp_net.requests_rejected", Type: datapoint.Counter},
	"asp_net.worker_process_restarts":                               {Name: "asp_net.worker_process_restarts", Type: datapoint.Counter},
	"asp_net.worker_processes_running":                              {Name: "asp_net.worker_processes_running"},
	"asp_net_applications.errors_during_execution":                  {Name: "asp_net_applications.errors_during_execution", Type: datapoint.Counter},
	"asp_net_applications.errors_total_persec":                      {Name: "asp_net_applications.errors_total_sec"},
	"asp_net_applications.errors_unhandled_during_execution_persec": {Name: "asp_net_applications.errors_unhandled_during_execution_sec"},
	"asp_net_applications.pipeline_instance_count":                  {Name: "asp_net_applications.pipeline_instance_count"},
	"asp_net_applications.requests_failed":                          {Name: "asp_net_applications.requests_failed", Type: datapoint.Counter},
	"asp_net_applications.requests_persec":                          {Name: "asp_net_applications.requests_sec"},
	"asp_net_applications.session_sql_server_connections_total":     {Name: "asp_net_applications.session_sql_server_connections_total"},
	"asp_net_applications.sessions_active":                          {Name: "asp_net_applications.sessions_active"},
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				ObjectName: "ASP.NET",
				Counters: []string{
					"applications running",
					"application restarts",
					"requests current",
					"requests queued",
					"requests rejected",
					"worker processes running",
					"worker process restarts",
				},
				Instances:     []string{"*"},
				Measurement:   "asp_net",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "ASP.NET Applications",
				Counters: []string{
					"requests failed",
					"requests/sec",
					"errors during execution",
					"errors unhandled during execution/sec",
					"errors total/sec",
					"pipeline instance count",
					"sessions active",
					"session sql server connections total",
				},
				Instances:     []string{"*"},
				Measurement:   "asp_net_applications",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
		},
	}

	plugin := winperfcounters.GetPlugin(perfcounterConf)

	// create batch emitter
	emitter := batchemitter.NewEmitter(m.Output, logger)

	// create the accumulator
	ac := accumulator.NewAccumulator(emitter)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}

		// process measurements retrieved from perf counter monitor
		for _, e := range perfhelper.ProcessMeasurements(emitter.Measurements, metricMap, m.Output.SendDatapoint, monitorType, "instance") {
			logger.Error(e.Error())
		}

		// reset batch emitter
		// NOTE: we can do this here because this emitter is on a single routine
		// if that changes, make sure you lock the mutex on the batch emitter
		emitter.Measurements = emitter.Measurements[:0]

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
