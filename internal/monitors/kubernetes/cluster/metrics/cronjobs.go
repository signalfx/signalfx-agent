package metrics

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
)

func datapointsForCronJob(cj *batchv1beta1.CronJob) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": cj.Namespace,
		"kubernetes_uid":       string(cj.UID),
		"kubernetes_name":      cj.Name,
	}

	return []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.cronjob.active",
			dimensions,
			datapoint.NewIntValue(int64(len(cj.Status.Active))),
			datapoint.Gauge,
			time.Now()),
	}
}

func dimPropsForCronJob(cj *batchv1beta1.CronJob) *atypes.DimProperties {
	props, tags := k8sutil.PropsAndTagsFromLabels(cj.Labels)

	props["kubernetes_workload"] = "CronJob"
	props["schedule"] = cj.Spec.Schedule
	props["concurrency_policy"] = string(cj.Spec.ConcurrencyPolicy)

	for _, or := range cj.OwnerReferences {
		props[utils.LowercaseFirstChar(or.Kind)] = or.Name
		props[utils.LowercaseFirstChar(or.Kind)+"_uid"] = string(or.UID)
	}

	if len(props) == 0 && len(tags) == 0 {
		return nil
	}

	return &atypes.DimProperties{
		Dimension: atypes.Dimension{
			Name:  "kubernetes_uid",
			Value: string(cj.UID),
		},
		Properties: props,
		Tags:       tags,
	}
}
