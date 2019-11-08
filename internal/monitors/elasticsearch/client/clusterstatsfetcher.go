package client

import (
	"github.com/signalfx/golib/v3/datapoint"
)

const (
	clusterStatusGreen  = "green"
	clusterStatusYellow = "yellow"
	clusterStatusRed    = "red"
)

// GetClusterStatsDatapoints fetches datapoints for ES cluster level stats
func GetClusterStatsDatapoints(clusterStatsOutput *ClusterStatsOutput, defaultDims map[string]string, enhanced bool) []*datapoint.Datapoint {
	var out []*datapoint.Datapoint

	if enhanced {
		out = append(out, []*datapoint.Datapoint{
			prepareGaugeHelper("elasticsearch.cluster.initializing-shards", defaultDims, clusterStatsOutput.InitializingShards),
			prepareGaugeHelper("elasticsearch.cluster.delayed-unassigned-shards", defaultDims, clusterStatsOutput.DelayedUnassignedShards),
			prepareGaugeHelper("elasticsearch.cluster.pending-tasks", defaultDims, clusterStatsOutput.NumberOfPendingTasks),
			prepareGaugeHelper("elasticsearch.cluster.in-flight-fetches", defaultDims, clusterStatsOutput.NumberOfInFlightFetch),
			prepareGaugeHelper("elasticsearch.cluster.task-max-wait-time", defaultDims, clusterStatsOutput.TaskMaxWaitingInQueueMillis),
			prepareGaugeFHelper("elasticsearch.cluster.active-shards-percent", defaultDims, clusterStatsOutput.ActiveShardsPercentAsNumber),
			prepareGaugeHelper("elasticsearch.cluster.status", defaultDims, getMetricValueFromClusterStatus(clusterStatsOutput.Status)),
		}...)
	}
	out = append(out, []*datapoint.Datapoint{
		prepareGaugeHelper("elasticsearch.cluster.active-primary-shards", defaultDims, clusterStatsOutput.ActivePrimaryShards),
		prepareGaugeHelper("elasticsearch.cluster.active-shards", defaultDims, clusterStatsOutput.ActiveShards),
		prepareGaugeHelper("elasticsearch.cluster.number-of-data_nodes", defaultDims, clusterStatsOutput.NumberOfDataNodes),
		prepareGaugeHelper("elasticsearch.cluster.number-of-nodes", defaultDims, clusterStatsOutput.NumberOfNodes),
		prepareGaugeHelper("elasticsearch.cluster.relocating-shards", defaultDims, clusterStatsOutput.RelocatingShards),
		prepareGaugeHelper("elasticsearch.cluster.unassigned-shards", defaultDims, clusterStatsOutput.UnassignedShards),
	}...)
	return out
}

// Map cluster status to a numeric value
func getMetricValueFromClusterStatus(s *string) *int64 {
	// For whatever reason if the monitor did not get cluster status return nil
	if s == nil {
		return nil
	}
	out := new(int64)
	status := *s

	switch status {
	case clusterStatusGreen:
		*out = 0
	case clusterStatusYellow:
		*out = 1
	case clusterStatusRed:
		*out = 2
	}

	return out
}
