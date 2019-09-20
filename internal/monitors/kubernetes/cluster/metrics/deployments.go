//nolint: dupl
package metrics

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
)

func datapointsForDeployment(dep *appsv1.Deployment) []*datapoint.Datapoint {
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

func dimPropsForDeployment(dep *appsv1.Deployment) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(dep.Labels)
	props["deployment"] = dep.Name
	props["kubernetes_workload"] = "Deployment"
	props["deployment_creation_timestamp"] = dep.GetCreationTimestamp().Format(time.RFC3339)

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
