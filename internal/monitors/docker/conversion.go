package docker

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	dtypes "github.com/docker/docker/api/types"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

func convertStatsToMetrics(container *dtypes.ContainerJSON, cstats *dtypes.ContainerStats) ([]*datapoint.Datapoint, error) {
	var parsed dtypes.StatsJSON
	err := json.NewDecoder(cstats.Body).Decode(&parsed)
	if err != nil {
		// EOF means that there aren't any stats, perhaps because the container
		// is gone.  Just return nothing and no error.
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	cstats.Body.Close()

	var dps []*datapoint.Datapoint
	dps = append(dps, convertBlkioStats(&parsed.BlkioStats)...)
	dps = append(dps, convertCPUStats(&parsed.CPUStats, &parsed.PreCPUStats)...)
	dps = append(dps, convertMemoryStats(&parsed.MemoryStats)...)
	dps = append(dps, convertNetworkStats(&parsed.Networks)...)

	now := time.Now()
	for i := range dps {
		dps[i].Timestamp = now

		if dps[i].Dimensions == nil {
			dps[i].Dimensions = map[string]string{}
		}
		// Set to preverse compatibility with docker-collectd plugin
		dps[i].Dimensions["plugin"] = "docker"
		name := strings.TrimPrefix(container.Name, "/")
		dps[i].Dimensions["container_name"] = name
		// Duplicate container_name in plugin_instance to maintain compat with
		// collectd-docker plugin
		dps[i].Dimensions["plugin_instance"] = name
		dps[i].Dimensions["container_image"] = container.Config.Image
		dps[i].Dimensions["container_id"] = container.ID
	}

	return dps, nil
}

func convertBlkioStats(stats *dtypes.BlkioStats) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	for k, v := range map[string][]dtypes.BlkioStatEntry{
		"io_service_bytes_recursive": stats.IoServiceBytesRecursive,
		"io_serviced_recursive":      stats.IoServicedRecursive,
		"io_queue_recursive":         stats.IoQueuedRecursive,
		"io_service_time_recursive":  stats.IoServiceTimeRecursive,
		"io_wait_time_recursive":     stats.IoWaitTimeRecursive,
		"io_merged_recursive":        stats.IoMergedRecursive,
		"io_time_recursive":          stats.IoTimeRecursive,
		"sectors_recursive":          stats.SectorsRecursive,
	} {
		for _, bs := range v {
			dims := map[string]string{
				"device_major": strconv.FormatUint(bs.Major, 10),
				"device_minor": strconv.FormatUint(bs.Minor, 10),
			}
			out = append(out, sfxclient.Cumulative("blkio."+k+"."+strings.ToLower(bs.Op), dims, int64(bs.Value)))
		}
	}
	return out
}

func convertCPUStats(stats *dtypes.CPUStats, prior *dtypes.CPUStats) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, []*datapoint.Datapoint{
		sfxclient.Cumulative("cpu.usage.total", nil, int64(stats.CPUUsage.TotalUsage)),
		sfxclient.Cumulative("cpu.usage.kernelmode", nil, int64(stats.CPUUsage.UsageInKernelmode)),
		sfxclient.Cumulative("cpu.usage.usermode", nil, int64(stats.CPUUsage.UsageInUsermode)),
		sfxclient.Cumulative("cpu.usage.system", nil, int64(stats.SystemUsage)),
	}...)

	for i, v := range stats.CPUUsage.PercpuUsage {
		dims := map[string]string{
			"core": "cpu" + strconv.Itoa(i),
		}
		out = append(out, sfxclient.Cumulative("cpu.percpu.usage", dims, int64(v)))
	}

	out = append(out, []*datapoint.Datapoint{
		sfxclient.Cumulative("cpu.throttling_data.periods", nil, int64(stats.ThrottlingData.Periods)),
		sfxclient.Cumulative("cpu.throttling_data.throttled_periods", nil, int64(stats.ThrottlingData.ThrottledPeriods)),
		sfxclient.Cumulative("cpu.throttling_data.throttled_time", nil, int64(stats.ThrottlingData.ThrottledTime)),
	}...)

	out = append(out, sfxclient.GaugeF("cpu.percent", nil, calculateCPUPercent(prior, stats)))

	return out
}

// Copied from
// https://github.com/docker/cli/blob/dbd96badb6959c2b7070664aecbcf0f7c299c538/cli/command/container/stats_helpers.go
func calculateCPUPercent(previous *dtypes.CPUStats, v *dtypes.CPUStats) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUUsage.TotalUsage) - float64(previous.CPUUsage.TotalUsage)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.SystemUsage) - float64(previous.SystemUsage)
		onlineCPUs  = float64(v.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(v.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func convertMemoryStats(stats *dtypes.MemoryStats) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	out = append(out, []*datapoint.Datapoint{
		sfxclient.Gauge("memory.usage.limit", nil, int64(stats.Limit)),
		sfxclient.Gauge("memory.usage.max", nil, int64(stats.MaxUsage)),
		sfxclient.Gauge("memory.usage.total", nil, int64(stats.Usage)),
		sfxclient.GaugeF("memory.percent", nil,
			// If cache is not present it will use the default value of 0
			100.0*(float64(stats.Usage)-float64(stats.Stats["cache"]))/float64(stats.Limit)),
	}...)

	for k, v := range stats.Stats {
		sfxclient.Gauge("memory.stats."+k, nil, int64(v))
	}

	return out
}

func convertNetworkStats(stats *map[string]dtypes.NetworkStats) []*datapoint.Datapoint {
	if stats == nil {
		return nil
	}
	var out []*datapoint.Datapoint
	for k, s := range *stats {
		dims := map[string]string{
			"interface": k,
		}

		out = append(out, []*datapoint.Datapoint{
			sfxclient.Cumulative("network.usage.rx_bytes", dims, int64(s.RxBytes)),
			sfxclient.Cumulative("network.usage.rx_dropped", dims, int64(s.RxDropped)),
			sfxclient.Cumulative("network.usage.rx_errors", dims, int64(s.RxErrors)),
			sfxclient.Cumulative("network.usage.rx_packets", dims, int64(s.RxPackets)),
			sfxclient.Cumulative("network.usage.tx_bytes", dims, int64(s.TxBytes)),
			sfxclient.Cumulative("network.usage.tx_dropped", dims, int64(s.TxDropped)),
			sfxclient.Cumulative("network.usage.tx_errors", dims, int64(s.TxErrors)),
			sfxclient.Cumulative("network.usage.tx_packets", dims, int64(s.TxPackets)),
		}...)
	}
	return out
}
