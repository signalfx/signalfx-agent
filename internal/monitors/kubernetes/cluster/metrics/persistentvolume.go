package metrics

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	v1 "k8s.io/api/core/v1"
)

func datapointsForPersistentVolume(pv *v1.PersistentVolume) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":   "kubernetes",
		"namespace":       pv.Namespace,
		"kubernetes_uid":  string(pv.UID),
		"kubernetes_name": pv.Name,
		"volume":          pv.Name, // to be able to link to data from kubernetes-volume monitor
	}

	dps := []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.persistent_volume_phase",
			dimensions,
			datapoint.NewIntValue(volumePhaseToInt(pv.Status.Phase)),
			datapoint.Gauge,
			time.Now()),
	}
	return dps
}

func dimPropsForPersistentVolume(pv *v1.PersistentVolume) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(pv.Labels)

	props["persistent_volume_creation_timestamp"] = pv.CreationTimestamp.Format(time.RFC3339)

	if pv.Spec.StorageClassName != "" {
		props["storage_class"] = pv.Spec.StorageClassName
	}

	for _, am := range pv.Spec.AccessModes {
		tags[string(am)] = true
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_uid",
			Value: string(pv.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}

func volumePhaseToInt(phase v1.PersistentVolumePhase) int64 {
	switch phase {
	case v1.VolumePending:
		return 1
	case v1.VolumeAvailable:
		return 2
	case v1.VolumeBound:
		return 3
	case v1.VolumeReleased:
		return 4
	case v1.VolumeFailed:
		return 5
	default:
		return 6
	}
}
