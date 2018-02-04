package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type replicationControllerMetrics struct {
	replicationControllers map[types.UID]replicaDPs
}

func newReplicationControllerMetrics() *replicationControllerMetrics {
	return &replicationControllerMetrics{
		replicationControllers: make(map[types.UID]replicaDPs),
	}
}

func (rcm replicationControllerMetrics) Add(obj runtime.Object) {
	rc := obj.(*v1.ReplicationController)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rc.Namespace,
		"uid":             string(rc.UID),
		"kubernetes_name": rc.Name,
	}

	if rc.Spec.Replicas == nil {
		return
	}
	rcm.replicationControllers[rc.UID] = makeReplicaDPs("replication_controller", dimensions,
		*rc.Spec.Replicas, rc.Status.AvailableReplicas)
}

func (rcm replicationControllerMetrics) Remove(obj runtime.Object) {
	rc := obj.(*v1.ReplicationController)
	delete(rcm.replicationControllers, rc.UID)
}

func (rcm *replicationControllerMetrics) Datapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	for _, rdps := range rcm.replicationControllers {
		dps = append(dps, rdps.Datapoints()...)
	}

	return dps
}
