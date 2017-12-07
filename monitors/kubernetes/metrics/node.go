package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

type nodeMetrics struct {
	nodes map[types.UID]*v1.Node
}

func newNodeMetrics() *nodeMetrics {
	return &nodeMetrics{
		nodes: make(map[types.UID]*v1.Node),
	}
}

func (n *nodeMetrics) Datapoints() []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, node := range n.nodes {
		dims := map[string]string{
			"host":      firstNodeHostname(node),
			"machineID": node.Status.NodeInfo.MachineID,
		}

		dps = append(dps, []*datapoint.Datapoint{
			sfxclient.Gauge("kubernetes.node_ready", dims, nodeConditionValue(node, v1.NodeReady)),
		}...)
	}
	return dps
}

func (n *nodeMetrics) Add(obj runtime.Object) {
	node := obj.(*v1.Node)
	n.nodes[node.UID] = node
}

func (n *nodeMetrics) Remove(obj runtime.Object) {
	node := obj.(*v1.Node)
	delete(n.nodes, node.UID)
}

var NodeConditionValues = map[v1.ConditionStatus]int64{
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
	return NodeConditionValues[status]
}

func firstNodeHostname(node *v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeHostName {
			return addr.Address
		}
	}
	return ""
}
