package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/api/extensions/v1beta1"
)

func datapointsForDeployment(dep *v1beta1.Deployment) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": dep.Namespace,
		"uid":             string(dep.UID),
		"kubernetes_name": dep.Name,
	}

	if dep.Spec.Replicas == nil { // || dep.Status.AvailableReplicas == nil {
		return nil
	}

	return makeReplicaDPs("deployment", dimensions,
		*dep.Spec.Replicas, dep.Status.AvailableReplicas)
}
