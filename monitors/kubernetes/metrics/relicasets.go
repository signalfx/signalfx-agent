package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

type replicaSetMetrics struct {
	replicaSets map[types.UID]replicaDPs
}

func newReplicaSetMetrics() *replicaSetMetrics {
	return &replicaSetMetrics{
		replicaSets: make(map[types.UID]replicaDPs),
	}
}

func (rsm *replicaSetMetrics) Add(obj runtime.Object) {
	rs := obj.(*v1beta1.ReplicaSet)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rs.Namespace,
		"uid":             string(rs.UID),
		"kubernetes_name": rs.Name,
	}

	if rs.Spec.Replicas == nil { //|| rs.Status.AvailableReplicas == nil {
		return
	}
	rsm.replicaSets[rs.UID] = makeReplicaDPs("replica_set", dimensions,
		*rs.Spec.Replicas, rs.Status.AvailableReplicas)
}

func (rsm *replicaSetMetrics) Remove(obj runtime.Object) {
	rs := obj.(*v1beta1.ReplicaSet)
	delete(rsm.replicaSets, rs.UID)
}

func (rsm *replicaSetMetrics) Datapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	for _, rdps := range rsm.replicaSets {
		dps = append(dps, rdps.Datapoints()...)
	}

	return dps
}
