package metrics

import (
	"fmt"

	"github.com/iancoleman/strcase"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	v1 "k8s.io/api/core/v1"
)

// A map to check for duplicate machine IDs
var machineIDToNodeNameMap = make(map[string]string)

func datapointsForNode(
	node *v1.Node,
	useNodeName bool,
	nodeConditionTypesToReport []string,
) []*datapoint.Datapoint {
	dims := map[string]string{
		"kubernetes_node": node.Name,
	}

	// If we aren't using the node name as the node id, then we need machine_id
	// to sync properties to.  Eventually we should just get rid of machine_id
	// if it doesn't become more reliable and dependable across k8s deployments.
	if !useNodeName {
		dims["machine_id"] = node.Status.NodeInfo.MachineID
	}

	datapoints := make([]*datapoint.Datapoint, 0)
	for _, nodeConditionTypeValue := range nodeConditionTypesToReport {
		nodeConditionMetric := fmt.Sprintf("kubernetes.node_%s", strcase.ToSnake(nodeConditionTypeValue))
		v1NodeConditionTypeValue := v1.NodeConditionType(nodeConditionTypeValue)
		datapoints = append(
			datapoints,
			sfxclient.Gauge(
				nodeConditionMetric, dims, nodeConditionValue(node, v1NodeConditionTypeValue),
			),
		)
	}
	return datapoints
}

func dimPropsForNode(node *v1.Node, useNodeName bool) *atypes.Dimension {
	props, tags := k8sutil.PropsAndTagsFromLabels(node.Labels)

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	dim := &atypes.Dimension{
		Name:  "kubernetes_node",
		Value: node.Name,
	}

	if !useNodeName {
		machineID := node.Status.NodeInfo.MachineID
		dim = &atypes.Dimension{
			Name:  "machine_id",
			Value: machineID,
		}

		if otherNodeName, ok := machineIDToNodeNameMap[machineID]; ok && otherNodeName != node.Name {
			logger.Errorf("Your K8s cluster appears to have duplicate node machine IDs, "+
				"node %s and %s both have machine ID %s.  Please set the `useNodeName` option "+
				"in this monitor config and set the top-level config option `sendMachineID` to "+
				"false.", node.Name, otherNodeName, machineID)
			return nil
		}

		machineIDToNodeNameMap[machineID] = node.Name
	}

	dim.Properties = props
	dim.Tags = tags
	return dim
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
