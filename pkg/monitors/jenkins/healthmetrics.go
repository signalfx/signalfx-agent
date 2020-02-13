package jenkins

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
)

// map of key: Metric-Name value: json-key
var healthMetricsMap = map[string]string{
	"jenkins.node.health.disk.space":      "disk-space",
	"jenkins.node.health.temporary.space": "temporary-space",
	"jenkins.node.health.plugins":         "plugins",
	"jenkins.node.health.thread-deadlock": "thread-deadlock",
}

type HealthMetrics map[string]map[string]interface{}

func healthMetrics(jkClient jc.JenkinsClient, metricsKey string) ([]*datapoint.Datapoint, error) {
	var health HealthMetrics
	err := jkClient.FetchJSON(fmt.Sprintf(healthEndpoint, metricsKey), &health)
	if err != nil {
		return nil, err
	}
	dps := getHealthMetricsDataPoints(health)
	return dps, nil
}

func getHealthMetricsDataPoints(health HealthMetrics) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, metric := range groupMetricsMap[groupHealth] {
		if jsonKey, ok := healthMetricsMap[metric]; ok {
			if val, ok := health[jsonKey]["healthy"]; ok {
				v, err := datapoint.CastMetricValueWithBool(val)
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
