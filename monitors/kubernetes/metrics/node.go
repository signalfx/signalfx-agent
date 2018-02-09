package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	atypes "github.com/signalfx/neo-agent/monitors/types"
	"k8s.io/api/core/v1"
)

func datapointsForNode(node *v1.Node) []*datapoint.Datapoint {
	dims := map[string]string{
		"host":      firstNodeHostname(node),
		"machineID": node.Status.NodeInfo.MachineID,
	}

	return []*datapoint.Datapoint{
		sfxclient.Gauge("kubernetes.node_ready", dims, nodeConditionValue(node, v1.NodeReady)),
	}
}

func dimPropsForNode(node *v1.Node) *atypes.DimProperties {
	props := make(map[string]string)

	for label, value := range node.Labels {
		props[propNameSanitizer.ReplaceAllLiteralString(label, "_")] = value
	}

	if len(props) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "machineID",
			Value: node.Status.NodeInfo.MachineID,
		},
		Properties: props,
	}
}

var nodeConditionValues = map[v1.ConditionStatus]int64{
	v1.ConditionTrue:    1,
	v1.ConditionFalse:   0,
	v1.ConditionUnknown: -1,
}

func nodeConditionValue(node *v1.Node, condType v1.NodeConditionType) int64 {
	status := v1.ConditionUnknown
	for _, c := range node.Status.Conditions {
		if c.Type == condType {
			status = c.Status
			break
		}
	}
	return nodeConditionValues[status]
}

func firstNodeHostname(node *v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeHostName {
			return addr.Address
		}
	}
	return ""
}
