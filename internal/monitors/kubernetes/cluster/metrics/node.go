package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"k8s.io/api/core/v1"
)

// GAUGE(kubernetes.node_ready): Whether this node is ready (1), not ready (0)
// or in an unknown state (-1)

// DIMENSION(machine_id): The machine ID from /etc/machine-id.  This should be
// unique across all nodes in your cluster, but some cluster deployment tools
// don't guarantee this.  This will not be sent if the `useNodeName` config
// option is set to true.

// DIMENSION(kubernetes_node): The name of the node, as defined by the `name`
// field of the node resource.

// PROPERTY(machine_id/kubernetes_node:<node label>): All non-blank labels on a
// given node will be synced as properties to the `machine_id` or
// `kubernetes_node` dimension value for that node.  Which dimension gets the
// properties is determined by the `useNodeName` config option.  Any blank
// values will be synced as tags on that same dimension.

// A map to check for duplicate machine IDs
var machineIDToNodeNameMap = make(map[string]string)

func datapointsForNode(node *v1.Node, useNodeName bool) []*datapoint.Datapoint {
	dims := map[string]string{
		"host":            firstNodeHostname(node),
		"kubernetes_node": node.Name,
	}

	// If we aren't using the node name as the node id, then we need machine_id
	// to sync properties to.  Eventually we should just get rid of machine_id
	// if it doesn't become more reliable and dependable across k8s deployments.
	if !useNodeName {
		dims["machine_id"] = node.Status.NodeInfo.MachineID
	}

	return []*datapoint.Datapoint{
		sfxclient.Gauge("kubernetes.node_ready", dims, nodeConditionValue(node, v1.NodeReady)),
	}
}

func dimPropsForNode(node *v1.Node, useNodeName bool) *atypes.DimProperties {
	props, tags := propsAndTagsFromLabels(node.Labels)

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	dim := atypes.Dimension{
		Name:  "kubernetes_node",
		Value: node.Name,
	}

	if !useNodeName {
		machineID := node.Status.NodeInfo.MachineID
		dim = atypes.Dimension{
			Name:  "machine_id",
			Value: machineID,
		}

		if otherNodeName, ok := machineIDToNodeNameMap[machineID]; ok && otherNodeName != node.Name {
			logger.Errorf("Your K8s cluster appears to have duplicate node machine IDs, "+
				"node %s and %s both have machine ID %s.  Please set the `useNodeName` option "+
				"in this monitor config and set the top-level config option `sendMachineID` to "+
				"false.", node.Name, otherNodeName, machineID)
		}

		machineIDToNodeNameMap[machineID] = node.Name
	}

	return &atypes.DimProperties{
		Dimension:  dim,
		Properties: props,
		Tags:       tags,
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
