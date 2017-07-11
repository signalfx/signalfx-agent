package converter

import (
	"regexp"
	"strconv"
	"time"

	"log"

	info "github.com/google/cadvisor/info/v1"
	"github.com/signalfx/golib/datapoint"
)

// This will usually be manager.Manager, but can be swapped out for testing.
type infoProvider interface {
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

// ContainerNameToLabelsFunc converter function
type ContainerNameToLabelsFunc func(containerName string) map[string]string

// CadvisorCollector metric collector and converter
type CadvisorCollector struct {
	infoProvider            infoProvider
	containerMetrics        []containerMetric
	containerSpecMetrics    []containerSpecMetric
	containerSpecMemMetrics []containerSpecMetric
	containerSpecCPUMetrics []containerSpecMetric
	containerNameToLabels   ContainerNameToLabelsFunc
	excludedImages          []*regexp.Regexp
	excludedNames           []*regexp.Regexp
	excludedLabels          [][]*regexp.Regexp
	machineInfoMetrics      []machineInfoMetric
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

func getContainerMetrics(excludedMetrics map[string]bool) []containerMetric {
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

	var index = 0
	for _, metric := range metrics {
		// check if metric is not on the exclusion list
		if _, ok := excludedMetrics[metric.name]; !ok {
			metrics[index] = metric
			index++
		}
	}

	// trim metrics down to the desired length
	metrics = metrics[:index]
	return metrics
}

func getContainerSpecCPUMetrics(excludedMetrics map[string]bool) []containerSpecMetric {
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
	}

	var index = 0
	for _, metric := range metrics {
		// check if metric is not on the exclusion list
		if _, ok := excludedMetrics[metric.name]; !ok {
			metrics[index] = metric
			index++
		}
	}
	// trim metrics down to the desired length
	metrics = metrics[:index]
	return metrics
}

func getContainerSpecMemMetrics(excludedMetrics map[string]bool) []containerSpecMetric {
	var metrics = []containerSpecMetric{
		{
			containerMetric: containerMetric{
				name:        "container_spec_memory_limit_bytes",
				help:        "",
				valueType:   datapoint.Gauge,
				extraLabels: []string{},
			},
			getValues: func(container *info.ContainerInfo) metricValues {
				return metricValues{{value: datapoint.NewIntValue(int64(container.Spec.Memory.Limit))}}
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

	var index = 0
	for _, metric := range metrics {
		// check if metric is not on the exclusion list
		if _, ok := excludedMetrics[metric.name]; !ok {
			metrics[index] = metric
			index++
		}
	}
	// trim metrics down to the desired length
	metrics = metrics[:index]
	return metrics
}

func getMachineInfoMetrics(excludedMetrics map[string]bool) []machineInfoMetric {
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
	var index = 0
	for _, metric := range metrics {
		// check if metric is not on the exclusion list
		if _, ok := excludedMetrics[metric.name]; !ok {
			metrics[index] = metric
			index++
		}
	}
	// trim metrics down to the desired length
	metrics = metrics[:index]
	return metrics
}

func getContainerSpecMetrics(excludedMetrics map[string]bool) []containerSpecMetric {
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

	var index = 0
	for _, metric := range metrics {
		// check if metric is not on the exclusion list
		if _, ok := excludedMetrics[metric.name]; !ok {
			metrics[index] = metric
			index++
		}
	}
	// trim metrics down to the desired length
	metrics = metrics[:index]
	return metrics
}

// NewCadvisorCollector creates new CadvisorCollector
func NewCadvisorCollector(infoProvider infoProvider, f ContainerNameToLabelsFunc, excludedImages []*regexp.Regexp, excludedNames []*regexp.Regexp, excludedLabels [][]*regexp.Regexp, excludedMetrics map[string]bool) *CadvisorCollector {
	return &CadvisorCollector{
		excludedImages:          excludedImages,
		excludedNames:           excludedNames,
		excludedLabels:          excludedLabels,
		infoProvider:            infoProvider,
		containerNameToLabels:   f,
		containerMetrics:        getContainerMetrics(excludedMetrics),
		containerSpecCPUMetrics: getContainerSpecCPUMetrics(excludedMetrics),
		containerSpecMemMetrics: getContainerSpecMemMetrics(excludedMetrics),
		containerSpecMetrics:    getContainerSpecMetrics(excludedMetrics),
		machineInfoMetrics:      getMachineInfoMetrics(excludedMetrics),
	}
}

// Collect fetches the stats from all containers and delivers them as
// Prometheus metrics. It implements prometheus.PrometheusCollector.
func (c *CadvisorCollector) Collect(ch chan<- datapoint.Datapoint) {
	c.collectMachineInfo(ch)
	c.collectVersionInfo(ch)
	c.collectContainersInfo(ch)
	//c.errors.Collect(ch)
}

func copyDims(dims map[string]string) map[string]string {
	newMap := make(map[string]string)
	for k, v := range dims {
		newMap[k] = v
	}
	return newMap
}

// isExcludedLabel - filters out containers if their labels match the excludedLabels regexp
func (c *CadvisorCollector) isExcludedLabel(container info.ContainerInfo) bool {
	var exlabel []*regexp.Regexp
	for _, exlabel = range c.excludedLabels {
		for label, value := range container.Spec.Labels {
			if exlabel[0].Match([]byte(label)) && exlabel[1].Match([]byte(value)) {
				return true
			}
		}
	}

	return false
}

// isExcludedImage - filters out containers if their image matches the excludedImages regexp
func (c *CadvisorCollector) isExcludedImage(container info.ContainerInfo) bool {
	var eximage *regexp.Regexp
	var image = []byte(container.Spec.Image)
	for _, eximage = range c.excludedImages {
		if eximage.Match(image) {
			return true
		}
	}
	return false
}

// isExcludedName - filters out containers if their name matches the excludedContainer regexp
func (c *CadvisorCollector) isExcludedName(container info.ContainerInfo) bool {
	var exname *regexp.Regexp
	var name = []byte(container.Name)
	for _, exname = range c.excludedNames {
		if exname.Match(name) {
			return true
		}
		for _, alias := range container.Aliases {
			var aliasBytes = []byte(alias)
			if exname.Match(aliasBytes) {
				return true
			}

		}
	}
	return false
}

// isExcluded - filters out containers if their name, images, or labels match the configured regexp filters
func (c *CadvisorCollector) isExcluded(container info.ContainerInfo) bool {
	return c.isExcludedImage(container) || c.isExcludedName(container) || c.isExcludedLabel(container)
}

func (c *CadvisorCollector) collectContainersInfo(ch chan<- datapoint.Datapoint) {
	containers, err := c.infoProvider.SubcontainersInfo("/")
	if err != nil {
		//c.errors.Set(1)
		log.Printf("Couldn't get containers: %s", err)
		return
	}
	for _, container := range containers {
		if c.isExcluded(container) {
			continue
		}
		dims := make(map[string]string)
		id := container.Name
		dims["id"] = id

		name := id
		if len(container.Aliases) > 0 {
			name = container.Aliases[0]
			dims["name"] = name
		}

		image := container.Spec.Image
		if len(image) > 0 {
			dims["image"] = image
		}

		if c.containerNameToLabels != nil {
			newLabels := c.containerNameToLabels(name)
			for k, v := range newLabels {
				dims[k] = v
			}
		}

		tt := time.Now()
		// Container spec
		for _, cm := range c.containerSpecMetrics {
			for _, metricValue := range cm.getValues(&container) {
				newDims := copyDims(dims)

				// Add extra dimensions
				for i, label := range cm.extraLabels {
					newDims[label] = metricValue.labels[i]
				}

				ch <- *datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt)
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

					ch <- *datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt)
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

					ch <- *datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt)
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

					ch <- *datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, stat.Timestamp)
				}
			}
		}
	}
}

func (c *CadvisorCollector) collectVersionInfo(ch chan<- datapoint.Datapoint) {}

func (c *CadvisorCollector) collectMachineInfo(ch chan<- datapoint.Datapoint) {
	machineInfo, err := c.infoProvider.GetMachineInfo()
	if err != nil {
		//c.errors.Set(1)
		log.Printf("Couldn't get machine info: %s", err)
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

			ch <- *datapoint.New(cm.name, newDims, metricValue.value, cm.valueType, tt)
		}
	}
}
