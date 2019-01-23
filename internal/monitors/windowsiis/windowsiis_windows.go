// +build windows

package windowsiis

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
	// Web Service PerfCounters
	// Connections
	"web_service.current_connections":        {Name: "web_service.current_connections"},
	"web_service.connection_attempts_persec": {Name: "web_service.connection_attempts_sec"},
	// Requests
	"web_service.post_requests_persec":         {Name: "web_service.post_requests_sec"},
	"web_service.get_requests_persec":          {Name: "web_service.get_requests_sec"},
	"web_service.total_method_requests_persec": {Name: "web_service.total_method_requests_sec"},
	// Bytes Transferred
	"web_service.bytes_received_persec": {Name: "web_service.bytes_received_sec"},
	"web_service.bytes_sent_persec":     {Name: "web_service.bytes_sent_sec"},
	// Files Transferred
	"web_service.files_received_persec": {Name: "web_service.files_received_sec"},
	"web_service.files_sent_persec":     {Name: "web_service.files_sent_sec"},
	// Not Found Errors
	"web_service.not_found_errors_persec": {Name: "web_service.not_found_errors_sec"},
	// Users
	"web_service.anonymous_users_persec":    {Name: "web_service.anonymous_users_sec"},
	"web_service.nonanonymous_users_persec": {Name: "web_service.nonanonymous_users_sec"},
	// Uptime
	"web_service.service_uptime": {Name: "web_service.service_uptime", Type: datapoint.Counter},
	// ISAPI requests
	"web_service.isapi_extension_requests_persec": {Name: "web_service.isapi_extension_requests_sec"},
	// Process PerfCounters
	"process.handle_count":           {Name: "process.handle_count"},
	"process.percent_processor_time": {Name: "process.pct_processor_time"},
	"process.id_process":             {Name: "process.id_process"},
	"process.private_bytes":          {Name: "process.private_bytes"},
	"process.thread_count":           {Name: "process.thread_count"},
	"process.virtual_bytes":          {Name: "process.virtual_bytes"},
	"process.working_set":            {Name: "process.working_set"},
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				ObjectName: "web service",
				Counters: []string{
					// Connections
					"current connections",     // Number of current connections to the web service
					"connection attempts/sec", // Rate that connections to web service are attempted
					// Requests
					"post requests/sec",         // Rate of HTTP POST requests
					"get requests/sec",          // Rate of HTTP GET requests
					"total method requests/sec", // Rate at which all HTTP requests are received
					// Bytes Transferred
					"bytes received/sec", // Rate that data is received by web service
					"bytes sent/sec",     // Rate that data is sent by web service
					// Files Transferred
					"files received/sec", // Rate at which files are received by web service
					"files sent/sec",     // Rate at which files are sent by web service
					// Not Found Errors
					"not found errors/sec", // Rate of 'Not Found' Errors
					// Users
					"anonymous users/sec",    // Rate at which users are making anonymous requests to the web service
					"nonanonymous users/sec", // Rate at which users are making nonanonymous requests to the web service
					// Uptime
					"service uptime", // Service uptime
					// ISAPI requests
					"isapi extension requests/sec", // Rate of ISAPI extension request processed simultaneously by the web service
				},
				Instances:     []string{"*"},
				Measurement:   "web_service",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "Process",
				Counters: []string{
					// The total number of handles currently open by this
					// process. This number is equal to the sum of the handles currently open by each thread in this process.
					"Handle Count",
					// The percentage of elapsed time that all process threads used the processor to execution instructions.
					// Code executed to handle some hardware interrupts and trap conditions are included in this count.
					"% Processor Time",
					// The unique identifier of this process. ID Process numbers are reused, so they only identify a process for the lifetime of that process.
					"ID Process",
					// The current size, in bytes, of memory that this process has allocated that cannot be shared with other processes.
					"Private Bytes",
					// The number of threads currently active in this process. Every running process has at least one thread.
					"Thread Count",
					// The current size, in bytes, of the virtual address space the process is using.
					// Use of virtual address space does not necessarily imply corresponding use of either disk or main memory pages.
					// Virtual space is finite, and the process can limit its ability to load libraries.
					"Virtual Bytes",
					// The current size, in bytes, of the Working Set of this process.
					// The Working Set is the set of memory pages touched recently by the threads in the process.
					// If free memory in the computer is above a threshold, pages are left in the Working Set of a process even if they are not in use.
					// When free memory falls below a threshold, pages are trimmed from Working Sets.
					// If they are needed, they will then be soft-faulted back into the Working Set before leaving main memory.
					"Working Set",
				},
				Instances:     []string{"w3wp"},
				Measurement:   "process",
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
