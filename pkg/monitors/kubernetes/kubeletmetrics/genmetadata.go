// Code generated by monitor-code-gen. DO NOT EDIT.

package kubeletmetrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

const monitorType = "kubelet-metrics"

const (
	groupPodEphemeralStats = "podEphemeralStats"
)

var groupSet = map[string]bool{
	groupPodEphemeralStats: true,
}

const (
	containerCPUUtilization          = "container_cpu_utilization"
	containerFsAvailableBytes        = "container_fs_available_bytes"
	containerFsCapacityBytes         = "container_fs_capacity_bytes"
	containerFsUsageBytes            = "container_fs_usage_bytes"
	containerMemoryAvailableBytes    = "container_memory_available_bytes"
	containerMemoryMajorPageFaults   = "container_memory_major_page_faults"
	containerMemoryPageFaults        = "container_memory_page_faults"
	containerMemoryRssBytes          = "container_memory_rss_bytes"
	containerMemoryUsageBytes        = "container_memory_usage_bytes"
	containerMemoryWorkingSetBytes   = "container_memory_working_set_bytes"
	podEphemeralStorageCapacityBytes = "pod_ephemeral_storage_capacity_bytes"
	podEphemeralStorageUsedBytes     = "pod_ephemeral_storage_used_bytes"
	podNetworkReceiveBytesTotal      = "pod_network_receive_bytes_total"
	podNetworkReceiveErrorsTotal     = "pod_network_receive_errors_total"
	podNetworkTransmitBytesTotal     = "pod_network_transmit_bytes_total"
	podNetworkTransmitErrorsTotal    = "pod_network_transmit_errors_total"
)

var metricSet = map[string]monitors.MetricInfo{
	containerCPUUtilization:          {Type: datapoint.Counter},
	containerFsAvailableBytes:        {Type: datapoint.Gauge},
	containerFsCapacityBytes:         {Type: datapoint.Gauge},
	containerFsUsageBytes:            {Type: datapoint.Gauge},
	containerMemoryAvailableBytes:    {Type: datapoint.Gauge},
	containerMemoryMajorPageFaults:   {Type: datapoint.Counter},
	containerMemoryPageFaults:        {Type: datapoint.Counter},
	containerMemoryRssBytes:          {Type: datapoint.Gauge},
	containerMemoryUsageBytes:        {Type: datapoint.Gauge},
	containerMemoryWorkingSetBytes:   {Type: datapoint.Gauge},
	podEphemeralStorageCapacityBytes: {Type: datapoint.Gauge, Group: groupPodEphemeralStats},
	podEphemeralStorageUsedBytes:     {Type: datapoint.Gauge, Group: groupPodEphemeralStats},
	podNetworkReceiveBytesTotal:      {Type: datapoint.Counter},
	podNetworkReceiveErrorsTotal:     {Type: datapoint.Counter},
	podNetworkTransmitBytesTotal:     {Type: datapoint.Counter},
	podNetworkTransmitErrorsTotal:    {Type: datapoint.Counter},
}

var defaultMetrics = map[string]bool{
	containerCPUUtilization:       true,
	containerFsAvailableBytes:     true,
	containerFsCapacityBytes:      true,
	containerFsUsageBytes:         true,
	containerMemoryUsageBytes:     true,
	podNetworkReceiveBytesTotal:   true,
	podNetworkReceiveErrorsTotal:  true,
	podNetworkTransmitBytesTotal:  true,
	podNetworkTransmitErrorsTotal: true,
}

var groupMetricsMap = map[string][]string{
	groupPodEphemeralStats: []string{
		podEphemeralStorageCapacityBytes,
		podEphemeralStorageUsedBytes,
	},
}

var monitorMetadata = monitors.Metadata{
	MonitorType:     "kubelet-metrics",
	DefaultMetrics:  defaultMetrics,
	Metrics:         metricSet,
	SendUnknown:     false,
	Groups:          groupSet,
	GroupMetricsMap: groupMetricsMap,
	SendAll:         false,
}
