package metrics

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/utils"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

// podMetrics keeps track of pod-related K8s metrics
type podMetrics struct {
	// Int value of the datapoint.Datapoint corresponds to the phase (see `phaseToInt` below)
	podPhases map[types.UID]*datapoint.Datapoint
	// These will cap at 5 according to
	// https://kubernetes.io/docs/api-reference/v1.5/#containerstatus-v1
	containerRestartCount map[ContainerID]*datapoint.Datapoint
}

func newPodMetrics() *podMetrics {
	return &podMetrics{
		podPhases:             make(map[types.UID]*datapoint.Datapoint),
		containerRestartCount: make(map[ContainerID]*datapoint.Datapoint),
	}
}

func (pm *podMetrics) Add(obj runtime.Object) {
	pod := obj.(*v1.Pod)
	dimensions := map[string]string{
		"metric_source": "kubernetes",
		// Try and be consistent with other plugin dimensions, despite
		// verbosity
		"kubernetes_namespace": pod.Namespace,
		"kubernetes_pod_uid":   string(pod.UID),
		"kubernetes_pod_name":  pod.Name,
		"host":                 pod.Spec.NodeName,
	}

	pm.podPhases[pod.UID] = datapoint.New(
		"kubernetes.pod_phase",
		dimensions,
		datapoint.NewIntValue(phaseToInt(pod.Status.Phase)),
		datapoint.Gauge,
		time.Now())

	for _, cs := range pod.Status.ContainerStatuses {
		contDims := utils.CloneStringMap(dimensions)
		contDims["container_id"] = strings.Replace(cs.ContainerID, "docker://", "", 1)
		contDims["container_spec_name"] = cs.Name
		contDims["container_image"] = cs.Image
		pm.containerRestartCount[makeContUID(pod.UID, cs.Name)] = datapoint.New(
			"kubernetes.container_restart_count",
			contDims,
			datapoint.NewIntValue(int64(cs.RestartCount)),
			datapoint.Gauge,
			time.Now())
	}
}

func (pm *podMetrics) Remove(obj runtime.Object) {
	pod := obj.(*v1.Pod)
	delete(pm.podPhases, pod.UID)
	for key := range pm.containerRestartCount {
		if strings.HasPrefix(string(key), string(pod.UID)+":") {
			delete(pm.containerRestartCount, key)
		}
	}
}

func (pm *podMetrics) Datapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	for _, dp := range pm.podPhases {
		dps = append(dps, dp)
	}

	for _, dp := range pm.containerRestartCount {
		dps = append(dps, dp)
	}

	return dps
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
