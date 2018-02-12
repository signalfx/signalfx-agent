package metrics

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
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
		"host":                 pod.Spec.NodeName,
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
	}

	return dps
}

func dimPropsForPod(pod *v1.Pod) *atypes.DimProperties {
	props, tags := propsAndTagsFromLabels(pod.Labels)

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_pod_uid",
			Value: string(pod.UID),
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
