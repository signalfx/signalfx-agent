package metrics

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func datapointsForPod(pod *v1.Pod) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source": "kubernetes",
		// Try and be consistent with other plugin dimensions, despite
		// verbosity
		"kubernetes_namespace": pod.Namespace,
		"kubernetes_pod_uid":   string(pod.UID),
		"kubernetes_pod_name":  pod.Name,
		"kubernetes_node":      pod.Spec.NodeName,
	}

	dps := []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.pod_phase",
			dimensions,
			datapoint.NewIntValue(phaseToInt(pod.Status.Phase)),
			datapoint.Gauge,
			time.Now()),
	}

	for _, cs := range pod.Status.ContainerStatuses {
		contDims := utils.CloneStringMap(dimensions)
		contDims["container_id"] = strings.Replace(cs.ContainerID, "docker://", "", 1)
		contDims["container_spec_name"] = cs.Name
		contDims["container_image"] = cs.Image

		dps = append(dps, datapoint.New(
			"kubernetes.container_restart_count",
			contDims,
			datapoint.NewIntValue(int64(cs.RestartCount)),
			datapoint.Gauge,
			time.Now()))

		dps = append(dps, datapoint.New(
			"kubernetes.container_ready",
			contDims,
			datapoint.NewIntValue(int64(utils.BoolToInt(cs.Ready))),
			datapoint.Gauge,
			time.Now()))
	}

	return dps
}

func dimPropsForPod(cachedPod *k8sutil.CachedPod, sc *k8sutil.ServiceCache,
	rsc *k8sutil.ReplicaSetCache) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(cachedPod.LabelSet)
	for _, or := range cachedPod.OwnerReferences {
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
		props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

	// if pod is selected by a service, sync service as a tag
	serviceTags := sc.GetMatchingServices(cachedPod)
	for _, tag := range serviceTags {
		tags["kubernetes_service_"+tag] = true
	}

	// if pod was created by a replicaSet, sync its deployment name as a property
	if replicaSetName, exists := props["replicaSet"]; exists {
		deploymentName, deploymentUID := rsc.GetMatchingDeployment(cachedPod.Namespace, replicaSetName)
		if deploymentName != nil {
			props["deployment"] = *deploymentName
			props["deployment_uid"] = string(deploymentUID)
		}
	}

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_pod_uid",
			Value: string(cachedPod.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}

func phaseToInt(phase v1.PodPhase) int64 {
	switch phase {
	case v1.PodPending:
		return 1
	case v1.PodRunning:
		return 2
	case v1.PodSucceeded:
		return 3
	case v1.PodFailed:
		return 4
	case v1.PodUnknown:
		return 5
	default:
		return 5
	}
}

// Container Id is not guaranteed to exist, so make our own
func makeContUID(podUID types.UID, name string) ContainerID {
	return ContainerID(string(podUID) + ":" + name)
}
