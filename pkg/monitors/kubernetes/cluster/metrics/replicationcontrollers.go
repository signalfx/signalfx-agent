package metrics

import (
	"github.com/signalfx/golib/v3/datapoint"
	v1 "k8s.io/api/core/v1"
)

func datapointsForReplicationController(rc *v1.ReplicationController) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rc.Namespace,
		"uid":                  string(rc.UID),
		"kubernetes_name":      rc.Name,
	}

	if rc.Spec.Replicas == nil {
		return nil
	}
	return makeReplicaDPs("replication_controller", dimensions,
		*rc.Spec.Replicas, rc.Status.AvailableReplicas)
}
