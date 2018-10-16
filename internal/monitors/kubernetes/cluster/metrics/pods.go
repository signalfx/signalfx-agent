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

// GAUGE(kubernetes.container_restart_count): How many times the container has
// restarted in the recent past.  This value is pulled directly from [the K8s
// API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#containerstatus-v1-core)
// and the value can go indefinitely high and be reset to 0 at any time
// depending on how your [kubelet is configured to prune dead
// containers](https://kubernetes.io/docs/concepts/cluster-administration/kubelet-garbage-collection/).
// It is best to not depend too much on the exact value but rather look at it
// as either `== 0`, in which case you can conclude there were no restarts in
// the recent past, or `> 0`, in which case you can conclude there were
// restarts in the recent past, and not try and analyze the value beyond that.

// GAUGE(kubernetes.pod_phase): Current phase of the pod (1 - Pending, 2 - Running, 3 - Succeeded, 4 - Failed, 5 - Unknown)
// GAUGE(kubernetes.container_ready): Whether a container has passed its readiness probe (0 for no, 1 for yes)

// PROPERTY(kubernetes_pod_uid:<pod label>): Any labels with non-blank values
// on the pod will be synced as properties to the `kubernetes_pod_uid`
// dimension. Any blank labels will be synced as tags on that same dimension.

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

func dimPropsForPod(pod *v1.Pod) *atypes.DimProperties {
	props, tags := propsAndTagsFromLabels(pod.Labels)

	for _, or := range pod.OwnerReferences {
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
        props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

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
