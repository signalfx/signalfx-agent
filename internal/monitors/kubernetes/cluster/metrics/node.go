package metrics

import (
	"reflect"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

// GAUGE(kubernetes.node_ready): Whether this node is ready (1), not ready (0)
// or in an unknown state (-1)

// DIMENSION(machine_id): The machine ID from /etc/machine-id.  This should be
// unique across all nodes in your cluster, but some cluster deployment tools
// don't guarantee this.

// PROPERTY(machine_id:<node label>): All non-blank labels on a given node will
// be synced as properties to the `machine_id` dimension value for that node.
// Any blank values will be synced as tags on that same dimension.

// A map to check for duplicate machine IDs
var machineIDToHostMap = make(map[string]string)

func datapointsForNode(node *v1.Node) []*datapoint.Datapoint {
	dims := map[string]string{
		"host":       firstNodeHostname(node),
		"machine_id": node.Status.NodeInfo.MachineID,
	}

	return []*datapoint.Datapoint{
		sfxclient.Gauge("kubernetes.node_ready", dims, nodeConditionValue(node, v1.NodeReady)),
	}
}

func dimPropsForNode(node *v1.Node) *atypes.DimProperties {
	props, tags := propsAndTagsFromLabels(node.Labels)

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	host := firstNodeHostname(node)
	machineID := node.Status.NodeInfo.MachineID

	if otherHost, ok := machineIDToHostMap[machineID]; ok && otherHost != host {
		log.Errorf("Your K8s cluster appears to have duplicate node machine IDs, "+
			"host %s and %s both have machine ID %s.  This is probably kubelet's fault.", host, otherHost, machineID)
		return nil
	}
	machineIDToHostMap[machineID] = host

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "machine_id",
			Value: machineID,
		},
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

// Nodes get updated a lot due to heartbeat checks that alter the
// lastHeartbeatCheck field within condition items.  Also the images can
// sometimes come in different orderings and we don't really care about them
// anyway, so just get rid of them before comparing.
func nodesDifferent(n1 *v1.Node, n2 *v1.Node) bool {
	c1 := *n1
	c2 := *n2

	c1.ResourceVersion = c2.ResourceVersion

	c1.Status.Conditions = nil
	c2.Status.Conditions = nil

	c1.Status.Images = nil
	c2.Status.Images = nil

	return !reflect.DeepEqual(c1, c2)
}
