package metrics

import (
	"time"

	"github.com/signalfx/golib/datapoint"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

// PodMetrics keeps track of pod-related K8s metrics
type daemonSetMetrics struct {
	daemonSets map[types.UID]daemonSetDPs
}

func newDaemonSetMetrics() *daemonSetMetrics {
	return &daemonSetMetrics{
		daemonSets: make(map[types.UID]daemonSetDPs),
	}
}

func (dsm *daemonSetMetrics) Add(obj runtime.Object) {
	ds := obj.(*v1beta1.DaemonSet)

	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": ds.Namespace,
		"uid":             string(ds.UID),
		"kubernetes_name": ds.Name,
	}
	dsm.daemonSets[ds.UID] = daemonSetDPs{
		CurrentNumberScheduled: datapoint.New(
			"kubernetes.daemon_set.current_scheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.CurrentNumberScheduled)),
			datapoint.Gauge,
			time.Now()),
		DesiredNumberScheduled: datapoint.New(
			"kubernetes.daemon_set.desired_scheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.DesiredNumberScheduled)),
			datapoint.Gauge,
			time.Now()),
		NumberMisscheduled: datapoint.New(
			"kubernetes.daemon_set.misscheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.NumberMisscheduled)),
			datapoint.Gauge,
			time.Now()),
		NumberReady: datapoint.New(
			"kubernetes.daemon_set.ready",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.NumberReady)),
			datapoint.Gauge,
			time.Now()),
	}
}
func (dsm *daemonSetMetrics) Remove(obj runtime.Object) {
	ds := obj.(*v1beta1.DaemonSet)
	delete(dsm.daemonSets, ds.UID)
}

func (dsm *daemonSetMetrics) Datapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	for _, dsdps := range dsm.daemonSets {
		dps = append(dps, dsdps.Datapoints()...)
	}

	return dps
}

// DaemonSetDPs hold datapoints relevant to Daemon Sets
type daemonSetDPs struct {
	CurrentNumberScheduled *datapoint.Datapoint
	DesiredNumberScheduled *datapoint.Datapoint
	NumberMisscheduled     *datapoint.Datapoint
	NumberReady            *datapoint.Datapoint
}

func (dsdps *daemonSetDPs) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		dsdps.CurrentNumberScheduled,
		dsdps.DesiredNumberScheduled,
		dsdps.NumberMisscheduled,
		dsdps.NumberReady,
	}
}
