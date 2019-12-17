package metrics

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors/kubernetes/cluster/meta"
	k8sutils "github.com/signalfx/signalfx-agent/pkg/monitors/kubernetes/utils"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
)

func datapointsForHpa(hpa *v2beta1.HorizontalPodAutoscaler) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": hpa.Namespace,
		"kubernetes_uid":       string(hpa.UID),
		"kubernetes_name":      hpa.Name,
	}

	return append([]*datapoint.Datapoint{
		datapoint.New(
			meta.KubernetesHpaSpecMaxReplicas,
			dimensions,
			datapoint.NewIntValue(int64(hpa.Spec.MaxReplicas)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			meta.KubernetesHpaSpecMinReplicas,
			dimensions,
			datapoint.NewIntValue(int64(*hpa.Spec.MinReplicas)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			meta.KubernetesHpaStatusCurrentReplicas,
			dimensions,
			datapoint.NewIntValue(int64(hpa.Status.CurrentReplicas)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			meta.KubernetesHpaStatusDesiredReplicas,
			dimensions,
			datapoint.NewIntValue(int64(hpa.Status.DesiredReplicas)),
			datapoint.Gauge,
			time.Now()),
	}, newStatusDatapoints(hpa, dimensions)...)
}

func dimensionForHpa(hpa *v2beta1.HorizontalPodAutoscaler) *types.Dimension {
	props, tags := k8sutils.PropsAndTagsFromLabels(hpa.Labels)

	for _, or := range hpa.OwnerReferences {
		props["kubernetes_workload"] = or.Kind
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
		props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

	return &types.Dimension{
		Name:       "kubernetes_uid",
		Value:      string(hpa.UID),
		Properties: props,
		Tags:       tags,
	}
}

func newStatusDatapoints(hpa *v2beta1.HorizontalPodAutoscaler, dimensions map[string]string) []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)
	for _, condition := range hpa.Status.Conditions {
		metric, value, err := newStatusMetricValue(condition)
		if err != nil {
			logger.WithError(err).Errorf("Could not create hpa status datapoint")
			continue
		}
		dps = append(dps, datapoint.New(metric, dimensions, value, datapoint.Gauge, time.Now()))
	}
	return dps
}

func newStatusMetricValue(condition v2beta1.HorizontalPodAutoscalerCondition) (metric string, value datapoint.Value, err error) {
	switch condition.Type {
	case v2beta1.ScalingActive:
		metric = meta.KubernetesHpaStatusConditionScalingActive
	case v2beta1.AbleToScale:
		metric = meta.KubernetesHpaStatusConditionAbleToScale
	case v2beta1.ScalingLimited:
		metric = meta.KubernetesHpaStatusConditionScalingLimited
	default:
		return "", nil, fmt.Errorf("invalid horizontal pod autoscaler condition type: %v", condition.Type)
	}
	switch condition.Status {
	case v1.ConditionTrue:
		value = datapoint.NewIntValue(1)
	case v1.ConditionFalse:
		value = datapoint.NewIntValue(0)
	case v1.ConditionUnknown:
		value = datapoint.NewIntValue(-1)
	default:
		return metric, nil, fmt.Errorf("invalid horizontal pod autoscaler condition status: %v", condition.Status)
	}
	return
}
