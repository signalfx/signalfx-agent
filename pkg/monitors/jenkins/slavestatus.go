package jenkins

import (
	"fmt"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/signalfx/golib/v3/datapoint"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
)

func slaveStatus(jkClient jc.JenkinsClient) ([]*datapoint.Datapoint, error) {
	nodes, err := jkClient.GoJenkins.GetAllNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to run GetAllNodes %v", err)
	}
	dps := getSlaveStatusDataPoints(nodes)

	return dps, nil
}

func getSlaveStatusDataPoints(nodes []*gojenkins.Node) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	metric := "jenkins.node.online.status"

	for _, node := range nodes {
		if node.Raw.Class == "hudson.slaves.SlaveComputer" {
			val, err := datapoint.CastMetricValueWithBool(!node.Raw.Offline)
			if err != nil {
				logger.Warnf("Skipping metric: failed to cast value for key %s and value %v : %v", metric, val, err)
			}
			dps = append(dps, datapoint.New(metric, map[string]string{"slave_name": node.Raw.DisplayName}, val, metricSet[metric].Type, time.Time{}))
		}
	}

	return dps
}
