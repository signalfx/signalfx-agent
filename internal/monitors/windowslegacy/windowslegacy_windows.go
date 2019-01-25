// +build windows

package windowslegacy

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/batchemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters/perfhelper"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var metricMap = map[string]*perfhelper.MetricMapper{
	// System PerfCounters
	// Processor
	"processor.percent_processor_time":  {Name: "processor.pct_processor_time"},
	"processor.percent_privileged_time": {Name: "processor.pct_privileged_time"},
	"processor.percent_user_time":       {Name: "processor.pct_user_time"},
	"processor.interrupts_persec":       {Name: "processor.interrupts_sec"},

	// System
	"system.processor_queue_length":  {Name: "system.processor_queue_length"},
	"system.system_calls_persec":     {Name: "system.system_calls_sec"},
	"system.context_switches_persec": {Name: "system.context_switches_sec"},

	// Memory
	"memory.available_mbytes":   {Name: "memory.available_mbytes"},
	"memory.pages_input_persec": {Name: "memory.pages_input_sec"},

	// Paging File
	"pagingfile.percent_usage":      {Name: "paging_file.pct_usage"},
	"pagingfile.percent_usage_peak": {Name: "paging_file.pct_usage_peak"},

	// PhysicalDisk
	"physicaldisk.avg._disk_sec/write":    {Name: "physicaldisk.avg_disk_sec_write"},
	"physicaldisk.avg._disk_sec/read":     {Name: "physicaldisk.avg_disk_sec_read"},
	"physicaldisk.avg._disk_sec/transfer": {Name: "physicaldisk.avg_disk_sec_transfer"},

	// Logical Disk
	"logicaldisk.disk_read_bytes_persec":  {Name: "logicaldisk.disk_read_bytes_sec"},
	"logicaldisk.disk_write_bytes_persec": {Name: "logicaldisk.disk_write_bytes_sec"},
	"logicaldisk.disk_transfers_persec":   {Name: "logicaldisk.disk_transfers_sec"},
	"logicaldisk.disk_reads_persec":       {Name: "logicaldisk.disk_reads_sec"},
	"logicaldisk.disk_writes_persec":      {Name: "logicaldisk.disk_writes_sec"},
	"logicaldisk.free_megabytes":          {Name: "logicaldisk.free_megabytes"},
	"logicaldisk.percent_free_space":      {Name: "logicaldisk.pct_free_space"},

	// Network
	"networkinterface.bytes_total_persec":         {Name: "network_interface.bytes_total_sec"},
	"networkinterface.bytes_received_persec":      {Name: "network_interface.bytes_received_sec"},
	"networkinterface.bytes_sent_persec":          {Name: "network_interface.bytes_sent_sec"},
	"networkinterface.current_bandwidth":          {Name: "network_interface.current_bandwidth"},
	"networkinterface.packets_received_persec":    {Name: "network_interface.packets_received_sec"},
	"networkinterface.packets_sent_persec":        {Name: "network_interface.packets_sent_sec"},
	"networkinterface.packets_received_errors":    {Name: "network_interface.packets_received_errors"},
	"networkinterface.packets_outbound_errors":    {Name: "network_interface.packets_outbound_errors"},
	"networkinterface.packets_received_discarded": {Name: "network_interface.received_discarded"},
	"networkinterface.packets_outbound_discarded": {Name: "network_interface.outbound_discarded"},
}

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	perfcounterConf := &winperfcounters.Config{
		CountersRefreshInterval: conf.CountersRefreshInterval,
		PrintValid:              conf.PrintValid,
		Object: []winperfcounters.PerfCounterObj{
			{
				ObjectName: "Processor",
				Counters: []string{
					"% Processor Time",
					"% Privileged Time",
					"% User Time",
					"Interrupts/sec",
				},
				Instances:     []string{"*"},
				Measurement:   "Processor",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "System",
				Counters: []string{
					"Processor Queue Length",
					"System Calls/sec",
					"Context Switches/sec",
				},
				Instances:     []string{"*"},
				Measurement:   "System",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "Memory",
				Counters: []string{
					"Available MBytes",
					"Pages Input/sec",
				},
				Instances:     []string{"*"},
				Measurement:   "Memory",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "Paging File",
				Counters: []string{
					"% Usage",
					"% Usage Peak",
				},
				Instances:     []string{"*"},
				Measurement:   "PagingFile",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "PhysicalDisk",
				Counters: []string{
					"Avg. Disk sec/Write",
					"Avg. Disk sec/Read",
					"Avg. Disk sec/Transfer",
				},
				Instances:     []string{"*"},
				Measurement:   "PhysicalDisk",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "LogicalDisk",
				Counters: []string{
					"Disk Read Bytes/sec",
					"Disk Write Bytes/sec",
					"Disk Transfers/sec",
					"Disk Reads/sec",
					"Disk Writes/sec",
					"Free Megabytes",
					"% Free Space",
				},
				Instances:     []string{"*"},
				Measurement:   "LogicalDisk",
				IncludeTotal:  true,
				WarnOnMissing: true,
			},
			{
				ObjectName: "Network Interface",
				Counters: []string{
					"Bytes Total/sec",
					"Bytes Received/sec",
					"Bytes Sent/sec",
					"Current Bandwidth",
					"Packets Received/sec",
					"Packets Sent/sec",
					"Packets Received Errors",
					"Packets Outbound Errors",
					"Packets Received Discarded",
					"Packets Outbound Discarded",
				},
				Instances:     []string{"*"},
				Measurement:   "NetworkInterface",
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
