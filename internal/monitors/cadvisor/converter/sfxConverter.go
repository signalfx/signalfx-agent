package converter

import (
	"math"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	info "github.com/google/cadvisor/info/v1"
	"github.com/signalfx/golib/datapoint"
)

// InfoProvider provides a swappable interface to actually get the cAdvisor
// metrics
type InfoProvider interface {
	// Get information about all subcontainers of the specified container (includes self).
	SubcontainersInfo(containerName string) ([]info.ContainerInfo, error)
	// Get information about the machine.
	GetMachineInfo() (*info.MachineInfo, error)
}

// metricValue describes a single metric value for a given set of label values
// within a parent containerMetric.
type metricValue struct {
	value  datapoint.Value
	labels []string
}

type metricValues []metricValue

type containerMetric struct {
	name        string
	help        string
	valueType   datapoint.MetricType
	extraLabels []string
	getValues   func(s *info.ContainerStats) metricValues
}

func (c *containerMetric) getName() string {
	return c.name
}

type containerSpecMetric struct {
	containerMetric
	getValues func(s *info.ContainerInfo) metricValues
}

type machineInfoMetric struct {
	containerMetric
	getValues func(s *info.MachineInfo) metricValues
}

// CadvisorCollector metric collector and converter
type CadvisorCollector struct {
	infoProvider            InfoProvider
	containerMetrics        []containerMetric
	containerSpecMetrics    []containerSpecMetric
	containerSpecMemMetrics []containerSpecMetric
	containerSpecCPUMetrics []containerSpecMetric
	machineInfoMetrics      []machineInfoMetric
	sendDP                  func(*datapoint.Datapoint)
	hostname                string
	defaultDimensions       map[string]string
}

// fsValues is a helper method for assembling per-filesystem stats.
func fsValues(fsStats []info.FsStats, valueFn func(*info.FsStats) datapoint.Value) metricValues {
	values := make(metricValues, 0, len(fsStats))
	for _, stat := range fsStats {
		values = append(values, metricValue{
			value:  valueFn(&stat),
			labels: []string{stat.Device},
		})
	}
	return values
}

func networkValues(net []info.InterfaceStats, valueFn func(*info.InterfaceStats) datapoint.Value) metricValues {
	values := make(metricValues, 0, len(net))
	for _, value := range net {
		values = append(values, metricValue{
			value:  valueFn(&value),
			labels: []string{value.Name},
		})
	}
	return values
}

// COUNTER(container_cpu_system_seconds_total): Cumulative system cpu time consumed in nanoseconds
// COUNTER(container_cpu_usage_seconds_total): Cumulative cpu time consumed per cpu in nanoseconds
// COUNTER(container_cpu_user_seconds_total): Cumulative user cpu time consumed in nanoseconds
// COUNTER(container_cpu_utilization): Cumulative cpu utilization in percentages
// COUNTER(container_cpu_cfs_periods): Total number of elapsed CFS enforcement intervals
// COUNTER(container_cpu_cfs_throttled_periods): Total number of times tasks in the cgroup have been throttled
// COUNTER(container_cpu_cfs_throttled_time): Total time duration, in nanoseconds, for which tasks in the cgroup have been throttled
// GAUGE(container_fs_io_current): Number of I/Os currently in progress
// COUNTER(container_fs_io_time_seconds_total): Cumulative count of seconds spent doing I/Os
// COUNTER(container_fs_io_time_weighted_seconds_total): Cumulative weighted I/O time in seconds
// GAUGE(container_fs_limit_bytes): Number of bytes that the container may occupy on this filesystem
// COUNTER(container_fs_read_seconds_total): Cumulative count of seconds spent reading
// COUNTER(container_fs_reads_merged_total): Cumulative count of reads merged
// COUNTER(container_fs_reads_total): Cumulative count of reads completed
// COUNTER(container_fs_sector_reads_total): Cumulative count of sector reads completed
// COUNTER(container_fs_sector_writes_total): Cumulative count of sector writes completed
// GAUGE(container_fs_usage_bytes): Number of bytes that are consumed by the container on this filesystem
// COUNTER(container_fs_write_seconds_total): Cumulative count of seconds spent writing
// COUNTER(container_fs_writes_merged_total): Cumulative count of writes merged
// COUNTER(container_fs_writes_total): Cumulative count of writes completed
// GAUGE(container_last_seen): Last time a container was seen by the exporter
// COUNTER(container_memory_failcnt): Number of memory usage hits limits
// COUNTER(container_memory_failures_total): Cumulative count of memory allocation failures
// GAUGE(container_memory_usage_bytes): Current memory usage in bytes
// GAUGE(container_memory_working_set_bytes): Current working set in bytes
// COUNTER(container_network_receive_bytes_total): Cumulative count of bytes received
// COUNTER(container_network_receive_errors_total): Cumulative count of errors encountered while receiving
// COUNTER(container_network_receive_packets_dropped_total): Cumulative count of packets dropped while receiving
// COUNTER(container_network_receive_packets_total): Cumulative count of packets received
// COUNTER(container_network_transmit_bytes_total): Cumulative count of bytes transmitted
// COUNTER(container_network_transmit_errors_total): Cumulative count of errors encountered while transmitting
// COUNTER(container_network_transmit_packets_dropped_total): Cumulative count of packets dropped while transmitting
// COUNTER(container_network_transmit_packets_total): Cumulative count of packets transmitted
// GAUGE(container_tasks_state): Number of tasks in given state

func getContainerMetrics() []containerMetric {
	var metrics = []containerMetric{
		{
			name:      "container_last_seen",
			help:      "Last time a container was seen by the exporter",
			valueType: datapoint.Timestamp,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(time.Now().UnixNano())}}
			},
		},
		{
			name:      "container_cpu_user_seconds_total",
			help:      "Cumulative user cpu time consumed in nanoseconds.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.Usage.User))}}
			},
		},
		{
			name:      "container_cpu_system_seconds_total",
			help:      "Cumulative system cpu time consumed in nanoseconds.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.Usage.System))}}
			},
		},
		{
			name:      "container_cpu_usage_seconds_total",
			help:      "Cumulative cpu time consumed per cpu in nanoseconds.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.Usage.Total))}}
			},
		},
		{
			name:      "container_cpu_utilization",
			help:      "Cumulative cpu utilization in percentages.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.Usage.Total / 10000000))}}
			},
		},
		{
			name:        "container_cpu_utilization_per_core",
			help:        "Cumulative cpu utilization in percentages per core",
			valueType:   datapoint.Counter,
			extraLabels: []string{"cpu"},
			getValues: func(s *info.ContainerStats) metricValues {
				metricValues := make(metricValues, len(s.Cpu.Usage.PerCpu))
				for index, coreUsage := range s.Cpu.Usage.PerCpu {
					if coreUsage > 0 {
						metricValues[index] = metricValue{value: datapoint.NewIntValue(int64(coreUsage / 10000000)), labels: []string{"cpu" + strconv.Itoa(index)}}
					} else {
						metricValues[index] = metricValue{value: datapoint.NewIntValue(int64(0)), labels: []string{strconv.Itoa(index)}}
					}
				}
				return metricValues
			},
		},
		{
			name:      "container_cpu_cfs_periods",
			help:      "Total number of elapsed CFS enforcement intervals.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.CFS.Periods))}}
			},
		},
		{
			name:      "container_cpu_cfs_throttled_periods",
			help:      "Total number of times tasks in the cgroup have been throttled",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.CFS.ThrottledPeriods))}}
			},
		},
		{
			name:      "container_cpu_cfs_throttled_time",
			help:      "Total time duration, in nanoseconds, for which tasks in the cgroup have been throttled.",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Cpu.CFS.ThrottledTime))}}
			},
		},
		{
			name:      "container_memory_failcnt",
			help:      "Number of memory usage hits limits",
			valueType: datapoint.Counter,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Memory.Failcnt))}}
			},
		},
		{
			name:      "container_memory_usage_bytes",
			help:      "Current memory usage in bytes.",
			valueType: datapoint.Gauge,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Memory.Usage))}}
			},
		},
		{
			name:      "container_memory_working_set_bytes",
			help:      "Current working set in bytes.",
			valueType: datapoint.Gauge,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(s.Memory.WorkingSet))}}
			},
		},
		{
			name:        "container_memory_failures_total",
			help:        "Cumulative count of memory allocation failures.",
			valueType:   datapoint.Counter,
			extraLabels: []string{"type", "scope"},
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{
					{
						value:  datapoint.NewIntValue(int64(s.Memory.ContainerData.Pgfault)),
						labels: []string{"pgfault", "container"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.Memory.ContainerData.Pgmajfault)),
						labels: []string{"pgmajfault", "container"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.Memory.HierarchicalData.Pgfault)),
						labels: []string{"pgfault", "hierarchy"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.Memory.HierarchicalData.Pgmajfault)),
						labels: []string{"pgmajfault", "hierarchy"},
					},
				}
			},
		},
		{
			name:        "container_fs_limit_bytes",
			help:        "Number of bytes that can be consumed by the container on this filesystem.",
			valueType:   datapoint.Gauge,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.Limit))
				})
			},
		},
		{
			name:        "container_fs_usage_bytes",
			help:        "Number of bytes that are consumed by the container on this filesystem.",
			valueType:   datapoint.Gauge,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.Usage))
				})
			},
		},
		{
			name:        "container_fs_reads_total",
			help:        "Cumulative count of reads completed",
			valueType:   datapoint.Gauge,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.ReadsCompleted))
				})
			},
		},
		{
			name:        "container_fs_sector_reads_total",
			help:        "Cumulative count of sector reads completed",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.SectorsRead))
				})
			},
		},
		{
			name:        "container_fs_reads_merged_total",
			help:        "Cumulative count of reads merged",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.ReadsMerged))
				})
			},
		},
		{
			name:        "container_fs_read_seconds_total",
			help:        "Cumulative count of seconds spent reading",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.ReadTime / uint64(time.Second)))
				})
			},
		},
		{
			name:        "container_fs_writes_total",
			help:        "Cumulative count of writes completed",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.WritesCompleted))
				})
			},
		},
		{
			name:        "container_fs_sector_writes_total",
			help:        "Cumulative count of sector writes completed",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.SectorsWritten))
				})
			},
		},
		{
			name:        "container_fs_writes_merged_total",
			help:        "Cumulative count of writes merged",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.WritesMerged))
				})
			},
		},
		{
			name:        "container_fs_write_seconds_total",
			help:        "Cumulative count of seconds spent writing",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.WriteTime / uint64(time.Second)))
				})
			},
		},
		{
			name:        "container_fs_io_current",
			help:        "Number of I/Os currently in progress",
			valueType:   datapoint.Gauge,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.IoInProgress))
				})
			},
		},
		{
			name:        "container_fs_io_time_seconds_total",
			help:        "Cumulative count of seconds spent doing I/Os",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.IoTime / uint64(time.Second)))
				})
			},
		},
		{
			name:        "container_fs_io_time_weighted_seconds_total",
			help:        "Cumulative weighted I/O time in seconds",
			valueType:   datapoint.Counter,
			extraLabels: []string{"device"},
			getValues: func(s *info.ContainerStats) metricValues {
				return fsValues(s.Filesystem, func(fs *info.FsStats) datapoint.Value {
					return datapoint.NewIntValue(int64(fs.WeightedIoTime / uint64(time.Second)))
				})
			},
		},
		{
			name:        "pod_network_receive_bytes_total",
			help:        "Cumulative count of bytes received",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.RxBytes))
				})
			},
		},
		{
			name:        "pod_network_receive_packets_total",
			help:        "Cumulative count of packets received",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.RxPackets))
				})
			},
		},
		{
			name:        "pod_network_receive_packets_dropped_total",
			help:        "Cumulative count of packets dropped while receiving",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.RxDropped))
				})
			},
		},
		{
			name:        "pod_network_receive_errors_total",
			help:        "Cumulative count of errors encountered while receiving",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.RxErrors))
				})
			},
		},
		{
			name:        "pod_network_transmit_bytes_total",
			help:        "Cumulative count of bytes transmitted",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.TxBytes))
				})
			},
		},
		{
			name:        "pod_network_transmit_packets_total",
			help:        "Cumulative count of packets transmitted",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.TxPackets))
				})
			},
		},
		{
			name:        "pod_network_transmit_packets_dropped_total",
			help:        "Cumulative count of packets dropped while transmitting",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.TxDropped))
				})
			},
		},
		{
			name:        "pod_network_transmit_errors_total",
			help:        "Cumulative count of errors encountered while transmitting",
			valueType:   datapoint.Counter,
			extraLabels: []string{"interface"},
			getValues: func(s *info.ContainerStats) metricValues {
				return networkValues(s.Network.Interfaces, func(is *info.InterfaceStats) datapoint.Value {
					return datapoint.NewIntValue(int64(is.TxErrors))
				})
			},
		},
		{
			name:        "container_tasks_state",
			help:        "Number of tasks in given state",
			extraLabels: []string{"state"},
			valueType:   datapoint.Gauge,
			getValues: func(s *info.ContainerStats) metricValues {
				return metricValues{
					{
						value:  datapoint.NewIntValue(int64(s.TaskStats.NrSleeping)),
						labels: []string{"sleeping"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.TaskStats.NrRunning)),
						labels: []string{"running"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.TaskStats.NrStopped)),
						labels: []string{"stopped"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.TaskStats.NrUninterruptible)),
						labels: []string{"uninterruptible"},
					},
					{
						value:  datapoint.NewIntValue(int64(s.TaskStats.NrIoWait)),
						labels: []string{"iowaiting"},
					},
				}
			},
		},
	}

	return metrics
}

// GAUGE(container_spec_cpu_shares): CPU share of the container

// GAUGE(container_spec_cpu_quota): In CPU quota for the CFS process scheduler.
// In K8s this is equal to the containers's CPU limit as a fraction of 1 core
// and multiplied by the `container_spec_cpu_period`.  So if the CPU limit is
// `500m` (500 millicores) for a container and the `container_spec_cpu_period`
// is set to 100,000, this value will be 50,000.

// GAUGE(container_spec_cpu_period): The number of microseconds that the [CFS
// scheduler](https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt)
// uses as a window when limiting container processes

func getContainerSpecCPUMetrics() []containerSpecMetric {
	var metrics = []containerSpecMetric{
		{
			containerMetric: containerMetric{
				name:        "container_spec_cpu_shares",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(container.Spec.Cpu.Limit))}}
			},
		},
		{
			containerMetric: containerMetric{
				name:        "container_spec_cpu_quota",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(container.Spec.Cpu.Quota))}}
			},
		},
		{
			containerMetric: containerMetric{
				name:        "container_spec_cpu_period",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(container.Spec.Cpu.Period))}}
			},
		},
	}

	return metrics
}

// GAUGE(container_spec_memory_limit_bytes): Memory limit for the container.
// GAUGE(container_spec_memory_swap_limit_bytes): Memory swap limit for the container.

func getContainerSpecMemMetrics() []containerSpecMetric {
	var metrics = []containerSpecMetric{
		{
			containerMetric: containerMetric{
				name:        "container_spec_memory_limit_bytes",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				limit := container.Spec.Memory.Limit
				if limit == math.MaxInt64 {
					limit = 0
				}
				return metricValues{{value: datapoint.NewIntValue(int64(limit))}}
			},
		},
		{
			containerMetric: containerMetric{
				name:        "container_spec_memory_swap_limit_bytes",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(container.Spec.Memory.SwapLimit))}}
			},
		},
	}

	return metrics
}

// GAUGE(machine_cpu_frequency_khz): Node's CPU frequency.
// GAUGE(machine_cpu_cores): Number of CPU cores on the node.
// GAUGE(machine_memory_bytes): Amount of memory installed on the node.

func getMachineInfoMetrics() []machineInfoMetric {
	var metrics = []machineInfoMetric{
		{
			containerMetric: containerMetric{
				name:        "machine_cpu_frequency_khz",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(machineInfo *info.MachineInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(machineInfo.CpuFrequency))}}
			},
		},
		{
			containerMetric: containerMetric{
				name:        "machine_cpu_cores",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(machineInfo *info.MachineInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(machineInfo.NumCores))}}
			},
		},
		{
			containerMetric: containerMetric{
				name:        "machine_memory_bytes",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(machineInfo *info.MachineInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(machineInfo.MemoryCapacity))}}
			},
		},
	}
	return metrics
}

// GAUGE(container_start_time_seconds): Start time of the container since unix epoch in seconds.

func getContainerSpecMetrics() []containerSpecMetric {
	var metrics = []containerSpecMetric{
		{
			containerMetric: containerMetric{
				name:        "container_start_time_seconds",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(container.Spec.CreationTime.Unix())}}
			},
		},
	}

	return metrics
}

// NewCadvisorCollector creates new CadvisorCollector
func NewCadvisorCollector(
	infoProvider InfoProvider,
	sendDP func(*datapoint.Datapoint),
	hostname string,
	defaultDimensions map[string]string) *CadvisorCollector {

	return &CadvisorCollector{
		infoProvider:            infoProvider,
		containerMetrics:        getContainerMetrics(),
		containerSpecCPUMetrics: getContainerSpecCPUMetrics(),
		containerSpecMemMetrics: getContainerSpecMemMetrics(),
		containerSpecMetrics:    getContainerSpecMetrics(),
		machineInfoMetrics:      getMachineInfoMetrics(),
		sendDP:                  sendDP,
		hostname:                hostname,
		defaultDimensions:       defaultDimensions,
	}
}

// Collect fetches the stats from all containers and delivers them as
// Prometheus metrics. It implements prometheus.PrometheusCollector.
func (c *CadvisorCollector) Collect() {
	c.collectMachineInfo()
	c.collectVersionInfo()
	c.collectContainersInfo()
}

func (c *CadvisorCollector) sendDatapoint(dp *datapoint.Datapoint) {
	dims := dp.Dimensions

	// filter POD level metrics
	if dims["container_name"] == "POD" {
		matched, _ := regexp.MatchString("^pod_network_.*", dp.Metric)
		if !matched {
			return
		}
		delete(dims, "container_name")
	}

	dims["metric_source"] = "kubernetes"
	dims["host"] = c.hostname

	for k, v := range c.defaultDimensions {
		dims[k] = v
	}

	// remove high cardinality dimensions
	delete(dims, "id")
	delete(dims, "name")

	c.sendDP(dp)
}

func copyDims(dims map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range dims {
		newMap[k] = v
	}
	return newMap
}

// DIMENSION(kubernetes_namespace): The K8s namespace the container is part of
// DIMENSION(kubernetes_pod_name): The pod instance under which this container runs
// DIMENSION(kubernetes_pod_uid): The UID of the pod instance under which this container runs
// DIMENSION(container_spec_name): The container's name as it appears in the pod spec
// DIMENSION(container_name): The container's name as it appears in the pod spec, the same as container_spec_name but retained for backwards compatibility.
// DIMENSION(container_id): The ID of the running container
// DIMENSION(container_image): The container image name

func (c *CadvisorCollector) collectContainersInfo() {
	containers, err := c.infoProvider.SubcontainersInfo("/")
	if err != nil {
		//c.errors.Set(1)
		log.WithError(err).Error("Couldn't get cAdvisor container stats")
		return
	}
	for _, container := range containers {
		dims := make(map[string]string)

		image := container.Spec.Image
		if len(image) > 0 {
			dims["container_image"] = image
		}

		dims["container_id"] = container.Id
		dims["container_spec_name"] = container.Spec.Labels["io.kubernetes.container.name"]
		// TODO: Remove this once everybody is migrated to neoagent v2 and
		// change built-in dashboards to use container_spec_name
		dims["container_name"] = container.Spec.Labels["io.kubernetes.container.name"]
		dims["kubernetes_pod_uid"] = container.Spec.Labels["io.kubernetes.pod.uid"]
		dims["kubernetes_pod_name"] = container.Spec.Labels["io.kubernetes.pod.name"]
		dims["kubernetes_namespace"] = container.Spec.Labels["io.kubernetes.pod.namespace"]

		tt := time.Now()
		// Container spec
		for _, cm := range c.containerSpecMetrics {
			for _, metricValue := range cm.getValues(&container) {
				newDims := copyDims(dims)

				// Add extra dimensions
				for i, label := range cm.extraLabels {
					newDims[label] = metricValue.labels[i]
				}

				c.sendDatapoint(datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt))
			}
		}

		if container.Spec.HasCpu {
			for _, cm := range c.containerSpecCPUMetrics {
				for _, metricValue := range cm.getValues(&container) {
					newDims := copyDims(dims)

					// Add extra dimensions
					for i, label := range cm.extraLabels {
						newDims[label] = metricValue.labels[i]
					}

					c.sendDatapoint(datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt))
				}
			}
		}

		if container.Spec.HasMemory {
			for _, cm := range c.containerSpecMemMetrics {
				for _, metricValue := range cm.getValues(&container) {
					newDims := copyDims(dims)

					// Add extra dimensions
					for i, label := range cm.extraLabels {
						newDims[label] = metricValue.labels[i]
					}

					c.sendDatapoint(datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt))
				}
			}
		}

		// Now for the actual metrics
		if len(container.Stats) > 0 {
			// only get the latest stats from this container. note/warning: the stats array contains historical statistics in earliest-to-latest order
			lastStatIndex := len(container.Stats) - 1
			stat := container.Stats[lastStatIndex]

			for _, cm := range c.containerMetrics {
				for _, metricValue := range cm.getValues(stat) {
					newDims := copyDims(dims)

					// Add extra dimensions
					for i, label := range cm.extraLabels {
						newDims[label] = metricValue.labels[i]
					}

					c.sendDatapoint(datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt))
				}
			}
		}
	}
}

func (c *CadvisorCollector) collectVersionInfo() {}

func (c *CadvisorCollector) collectMachineInfo() {
	machineInfo, err := c.infoProvider.GetMachineInfo()
	if err != nil {
		//c.errors.Set(1)
		log.Errorf("Couldn't get machine info: %s", err)
		return
	}
	if machineInfo == nil {
		return
	}

	dims := make(map[string]string)
	tt := time.Now()

	for _, cm := range c.machineInfoMetrics {
		for _, metricValue := range cm.getValues(machineInfo) {
			newDims := copyDims(dims)

			// Add extra dimensions
			for i, label := range cm.extraLabels {
				newDims[label] = metricValue.labels[i]
			}

			c.sendDatapoint(datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt))
		}
	}
}
