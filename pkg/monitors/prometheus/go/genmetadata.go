// Code generated by monitor-code-gen. DO NOT EDIT.

package prometheusgo

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

const monitorType = "prometheus/go"

var groupSet = map[string]bool{}

const (
	goGcDurationSeconds          = "go_gc_duration_seconds"
	goGcDurationSecondsBucket    = "go_gc_duration_seconds_bucket"
	goGcDurationSecondsCount     = "go_gc_duration_seconds_count"
	goGoroutines                 = "go_goroutines"
	goInfo                       = "go_info"
	goMemstatsAllocBytes         = "go_memstats_alloc_bytes"
	goMemstatsAllocBytesTotal    = "go_memstats_alloc_bytes_total"
	goMemstatsBuckHashSysBytes   = "go_memstats_buck_hash_sys_bytes"
	goMemstatsFreesTotal         = "go_memstats_frees_total"
	goMemstatsGcCPUFraction      = "go_memstats_gc_cpu_fraction"
	goMemstatsGcSysBytes         = "go_memstats_gc_sys_bytes"
	goMemstatsHeapAllocBytes     = "go_memstats_heap_alloc_bytes"
	goMemstatsHeapIdleBytes      = "go_memstats_heap_idle_bytes"
	goMemstatsHeapInuseBytes     = "go_memstats_heap_inuse_bytes"
	goMemstatsHeapObjects        = "go_memstats_heap_objects"
	goMemstatsHeapReleasedBytes  = "go_memstats_heap_released_bytes"
	goMemstatsHeapSysBytes       = "go_memstats_heap_sys_bytes"
	goMemstatsLastGcTimeSeconds  = "go_memstats_last_gc_time_seconds"
	goMemstatsLookupsTotal       = "go_memstats_lookups_total"
	goMemstatsMallocsTotal       = "go_memstats_mallocs_total"
	goMemstatsMcacheInuseBytes   = "go_memstats_mcache_inuse_bytes"
	goMemstatsMcacheSysBytes     = "go_memstats_mcache_sys_bytes"
	goMemstatsMspanInuseBytes    = "go_memstats_mspan_inuse_bytes"
	goMemstatsMspanSysBytes      = "go_memstats_mspan_sys_bytes"
	goMemstatsNextGcBytes        = "go_memstats_next_gc_bytes"
	goMemstatsOtherSysBytes      = "go_memstats_other_sys_bytes"
	goMemstatsStackInuseBytes    = "go_memstats_stack_inuse_bytes"
	goMemstatsStackSysBytes      = "go_memstats_stack_sys_bytes"
	goMemstatsSysBytes           = "go_memstats_sys_bytes"
	goThreads                    = "go_threads"
	processCPUSecondsTotal       = "process_cpu_seconds_total"
	processMaxFds                = "process_max_fds"
	processOpenFds               = "process_open_fds"
	processResidentMemoryBytes   = "process_resident_memory_bytes"
	processStartTimeSeconds      = "process_start_time_seconds"
	processVirtualMemoryBytes    = "process_virtual_memory_bytes"
	processVirtualMemoryMaxBytes = "process_virtual_memory_max_bytes"
)

var metricSet = map[string]monitors.MetricInfo{
	goGcDurationSeconds:          {Type: datapoint.Counter},
	goGcDurationSecondsBucket:    {Type: datapoint.Counter},
	goGcDurationSecondsCount:     {Type: datapoint.Counter},
	goGoroutines:                 {Type: datapoint.Gauge},
	goInfo:                       {Type: datapoint.Gauge},
	goMemstatsAllocBytes:         {Type: datapoint.Gauge},
	goMemstatsAllocBytesTotal:    {Type: datapoint.Counter},
	goMemstatsBuckHashSysBytes:   {Type: datapoint.Gauge},
	goMemstatsFreesTotal:         {Type: datapoint.Counter},
	goMemstatsGcCPUFraction:      {Type: datapoint.Gauge},
	goMemstatsGcSysBytes:         {Type: datapoint.Gauge},
	goMemstatsHeapAllocBytes:     {Type: datapoint.Gauge},
	goMemstatsHeapIdleBytes:      {Type: datapoint.Gauge},
	goMemstatsHeapInuseBytes:     {Type: datapoint.Gauge},
	goMemstatsHeapObjects:        {Type: datapoint.Gauge},
	goMemstatsHeapReleasedBytes:  {Type: datapoint.Gauge},
	goMemstatsHeapSysBytes:       {Type: datapoint.Gauge},
	goMemstatsLastGcTimeSeconds:  {Type: datapoint.Gauge},
	goMemstatsLookupsTotal:       {Type: datapoint.Counter},
	goMemstatsMallocsTotal:       {Type: datapoint.Counter},
	goMemstatsMcacheInuseBytes:   {Type: datapoint.Gauge},
	goMemstatsMcacheSysBytes:     {Type: datapoint.Gauge},
	goMemstatsMspanInuseBytes:    {Type: datapoint.Gauge},
	goMemstatsMspanSysBytes:      {Type: datapoint.Gauge},
	goMemstatsNextGcBytes:        {Type: datapoint.Gauge},
	goMemstatsOtherSysBytes:      {Type: datapoint.Gauge},
	goMemstatsStackInuseBytes:    {Type: datapoint.Gauge},
	goMemstatsStackSysBytes:      {Type: datapoint.Gauge},
	goMemstatsSysBytes:           {Type: datapoint.Gauge},
	goThreads:                    {Type: datapoint.Gauge},
	processCPUSecondsTotal:       {Type: datapoint.Counter},
	processMaxFds:                {Type: datapoint.Gauge},
	processOpenFds:               {Type: datapoint.Gauge},
	processResidentMemoryBytes:   {Type: datapoint.Gauge},
	processStartTimeSeconds:      {Type: datapoint.Gauge},
	processVirtualMemoryBytes:    {Type: datapoint.Gauge},
	processVirtualMemoryMaxBytes: {Type: datapoint.Gauge},
}

var defaultMetrics = map[string]bool{
	processStartTimeSeconds: true,
}

var groupMetricsMap = map[string][]string{}

var monitorMetadata = monitors.Metadata{
	MonitorType:       "prometheus/go",
	DefaultMetrics:    defaultMetrics,
	Metrics:           metricSet,
	MetricsExhaustive: false,
	Groups:            groupSet,
	GroupMetricsMap:   groupMetricsMap,
	SendAll:           false,
}
