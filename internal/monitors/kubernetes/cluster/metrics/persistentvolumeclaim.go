package metrics

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	v1 "k8s.io/api/core/v1"
)

func datapointsForPersistentVolumeClaim(pvc *v1.PersistentVolumeClaim) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":   "kubernetes",
		"namespace":       pvc.Namespace,
		"kubernetes_uid":  string(pvc.UID),
		"kubernetes_name": pvc.Name,
	}

	dps := []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.persistent_volume_claim_phase",
			dimensions,
			datapoint.NewIntValue(volumeClaimPhaseToInt(pvc.Status.Phase)),
			datapoint.Gauge,
			time.Now()),
	}

	//TODO: (Akash) Get resource limit and request stats
	return dps
}

func dimPropsForPersistentVolumeClaim(pvc *v1.PersistentVolumeClaim) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(pvc.Labels)

	props["persistent_volume_claim_creation_timestamp"] = pvc.CreationTimestamp.Format(time.RFC3339)

	for _, am := range pvc.Spec.AccessModes {
		tags[string(am)] = true
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_uid",
			Value: string(pvc.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}

func volumeClaimPhaseToInt(phase v1.PersistentVolumeClaimPhase) int64 {
	switch phase {
	case v1.ClaimPending:
		return 1
	case v1.ClaimBound:
		return 2
	case v1.ClaimLost:
		return 3
	default:
		return 4
	}
}
