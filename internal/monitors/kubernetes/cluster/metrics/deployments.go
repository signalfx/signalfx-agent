package metrics

import (
	"github.com/signalfx/golib/datapoint"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
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
		"kubernetes_uid":       string(dep.UID),
		"kubernetes_name":      dep.Name,
	}

	if dep.Spec.Replicas == nil { // || dep.Status.AvailableReplicas == nil {
		return nil
	}

	return makeReplicaDPs("deployment", dimensions,
		*dep.Spec.Replicas, dep.Status.AvailableReplicas)
}

func dimPropsForDeployment(dep *v1beta1.Deployment) *atypes.DimProperties {
	props, tags := propsAndTagsFromLabels(dep.Labels)
	props["name"] = dep.Name

	for _, or := range dep.OwnerReferences {
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
		props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_uid",
			Value: string(dep.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}
