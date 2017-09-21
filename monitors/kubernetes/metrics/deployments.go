package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

type deploymentMetrics struct {
	deployments map[types.UID]replicaDPs
}

func newDeploymentMetrics() *deploymentMetrics {
	return &deploymentMetrics{
		deployments: make(map[types.UID]replicaDPs),
	}
}

func (dm *deploymentMetrics) Add(obj runtime.Object) {
	dep := obj.(*v1beta1.Deployment)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": dep.Namespace,
		"uid":             string(dep.UID),
		"kubernetes_name": dep.Name,
	}

	if dep.Spec.Replicas == nil { // || dep.Status.AvailableReplicas == nil {
		return
	}
	dm.deployments[dep.UID] = makeReplicaDPs("deployment", dimensions,
		*dep.Spec.Replicas, dep.Status.AvailableReplicas)
}

func (dm *deploymentMetrics) Remove(obj runtime.Object) {
	d := obj.(*v1beta1.Deployment)
	delete(dm.deployments, d.UID)
}

func (dm *deploymentMetrics) Datapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	for _, rdps := range dm.deployments {
		dps = append(dps, rdps.Datapoints()...)
	}

	return dps
}
