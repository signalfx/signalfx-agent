package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/api/extensions/v1beta1"
)

func datapointsForReplicaSet(rs *v1beta1.ReplicaSet) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rs.Namespace,
		"uid":             string(rs.UID),
		"kubernetes_name": rs.Name,
	}

	if rs.Spec.Replicas == nil { //|| rs.Status.AvailableReplicas == nil {
		return nil
	}
	return makeReplicaDPs("replica_set", dimensions,
		*rs.Spec.Replicas, rs.Status.AvailableReplicas)
}
