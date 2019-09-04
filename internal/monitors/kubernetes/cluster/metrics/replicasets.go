//nolint: dupl
package metrics

import (
	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"k8s.io/api/extensions/v1beta1"
)

func datapointsForReplicaSet(rs *v1beta1.ReplicaSet) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rs.Namespace,
		"kubernetes_uid":       string(rs.UID),
		"kubernetes_name":      rs.Name,
	}

	if rs.Spec.Replicas == nil { //|| rs.Status.AvailableReplicas == nil {
		return nil
	}
	return makeReplicaDPs("replica_set", dimensions,
		*rs.Spec.Replicas, rs.Status.AvailableReplicas)
}

func dimPropsForReplicaSet(rs *v1beta1.ReplicaSet) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(rs.Labels)
	props["name"] = rs.Name
	props["k8s_workload"] = "ReplicaSet"

	for _, or := range rs.OwnerReferences {
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
		props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_uid",
			Value: string(rs.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}
