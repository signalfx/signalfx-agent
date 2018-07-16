package metrics

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"k8s.io/api/core/v1"
)

// GAUGE(kubernetes.resource_quota_used): The usage for a particular resource
// in a specific namespace.  Will only be sent if a quota is specified.

// GAUGE(kubernetes.resource_quota_hard): The upper limit for a particular
// resource in a specific namespace.  Will only be sent if a quota is
// specified.

// DIMENSION(resource): The k8s resource that the quota applies to
// DIMENSION(quota_name): The name of the k8s ResourceQuota object that the
// quota is part of

func datapointsForResourceQuota(rq *v1.ResourceQuota) []*datapoint.Datapoint {
	dps := []*datapoint.Datapoint{}

	for _, t := range []struct {
		typ string
		rl  v1.ResourceList
	}{
		{
			"hard",
			rq.Status.Hard,
		},
		{
			"used",
			rq.Status.Used,
		},
	} {
		for k, v := range t.rl {
			dims := map[string]string{
				"resource":             string(k),
				"quota_name":           rq.Name,
				"kubernetes_namespace": rq.Namespace,
			}

			asInt, ok := v.AsInt64()
			if !ok {
				asInt = v.AsDec().UnscaledBig().Int64()
			}
			dps = append(dps, sfxclient.Gauge("kubernetes.resource_quota_"+t.typ, dims, asInt))
		}
	}
	return dps
}
