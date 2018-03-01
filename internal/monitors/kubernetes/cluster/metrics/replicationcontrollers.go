package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/api/core/v1"
)

// GAUGE(kubernetes.replication_controller.available): Total number of
// available pods (ready for at least minReadySeconds) targeted by this
// replication controller.

// GAUGE(kubernetes.replication_controller.desired): Number of desired pods

func datapointsForReplicationController(rc *v1.ReplicationController) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rc.Namespace,
		"uid":             string(rc.UID),
		"kubernetes_name": rc.Name,
	}

	if rc.Spec.Replicas == nil {
		return nil
	}
	return makeReplicaDPs("replication_controller", dimensions,
		*rc.Spec.Replicas, rc.Status.AvailableReplicas)
}
