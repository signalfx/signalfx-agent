package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/api/extensions/v1beta1"
)

// GAUGE(kubernetes.deployment.available): Total number of available pods
// (ready for at least minReadySeconds) targeted by this deployment.

// GAUGE(kubernetes.deployment.desired): Number of desired pods in this
// deployment

func datapointsForDeployment(dep *v1beta1.Deployment) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": dep.Namespace,
		"kubernetes_deployment_uid":             string(dep.UID),
		"kubernetes_name": dep.Name,
	}

	if dep.Spec.Replicas == nil { // || dep.Status.AvailableReplicas == nil {
		return nil
	}

	return makeReplicaDPs("deployment", dimensions,
		*dep.Spec.Replicas, dep.Status.AvailableReplicas)
}
