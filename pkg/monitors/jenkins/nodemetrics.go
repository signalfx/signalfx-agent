package jenkins

import (
	"fmt"
	"strings"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
)

// map key:metric_name value:json_key
var nodeMetricsMap = map[string]string{
	"jenkins.node.vm.memory.total.used":    "vm.memory.total.used",
	"jenkins.node.vm.memory.heap.usage":    "vm.memory.non-heap.used",
	"jenkins.node.vm.memory.non-heap.used": "vm.memory.non-heap.used",
	"jenkins.node.queue.size":              "jenkins.queue.size.value",
	"jenkins.node.health-check.score":      "jenkins.health-check.score",
	"jenkins.node.executor.count":          "jenkins.executor.count.value",
	"jenkins.node.executor.in-use":         "jenkins.executor.in-use.value",
}

type NodeMetrics struct {
	Version string                            `json:"version"`
	Gauges  map[string]map[string]interface{} `json:"gauges"`
}

func nodeMetrics(jkClient jc.JenkinsClient, metricsKey string) ([]*datapoint.Datapoint, error) {
	var node NodeMetrics
	err := jkClient.FetchJSON(fmt.Sprintf(metricsEndpoint, metricsKey), &node)
	if err != nil {
		return nil, err
	}
	dps := getNodeMetricsDataPoints(&node)
	return dps, nil
}

func getNodeMetricsDataPoints(node *NodeMetrics) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, metric := range groupMetricsMap[groupNode] {
		if jsonKey, ok := nodeMetricsMap[metric]; ok {
			if val, ok := node.Gauges[jsonKey]; ok {
				v, err := datapoint.CastMetricValue(val["value"])
				if err != nil {
					// Could be a string
					logger.Warnf("Skipping metric: failed to cast value for key %s and value %v : %v", metric, val, err)
					continue
				}
				dps = append(dps, datapoint.New(metric, nil, v, metricSet[metric].Type, time.Time{}))
			} else {
				logger.Warnf("Failed to find json key for metric: %s", metric)
			}
		}
	}
	return dps
}

func liveness(jkClient jc.JenkinsClient, metricsKey string) []*datapoint.Datapoint {
	var val int64
	result, err := jkClient.FetchText(fmt.Sprintf(pingEndPoint, metricsKey))
	if err != nil {
		logger.WithError(err).Warnf("Failed to ping the jenkins server, sending status 0")
		val = 0
	}
	if strings.TrimSpace(result) == "pong" {
		val = 1
	}
	var dps []*datapoint.Datapoint
	dps = append(dps, datapoint.New(jenkinsNodeOnlineStatus, nil, datapoint.NewIntValue(val), metricSet[jenkinsNodeOnlineStatus].Type, time.Time{}))

	return dps
}

func getUniqueDimension(jkClient jc.JenkinsClient, jenkinsCluster string) (*types.Dimension, error) {
	nodeInfo, err := jkClient.GoJenkins.Info()
	if err != nil {
		return nil, err
	}

	dim := &types.Dimension{
		Name:              "jenkins_cluster",
		Value:             jenkinsCluster,
		Tags:              map[string]bool{},
		MergeIntoExisting: true,
		Properties:        map[string]string{"jenkins_version": jkClient.GoJenkins.Version},
	}

	for _, labelMap := range nodeInfo.AssignedLabels {
		if label, ok := labelMap["name"]; ok {
			// Every jenkins master has master as label, so skip it
			if label == "master" {
				continue
			}
			dim.Tags[label] = true
		}
	}

	return dim, nil
}
