package metrics

import (
	"time"

	"k8s.io/api/extensions/v1beta1"

	"github.com/signalfx/golib/datapoint"
)

// GAUGE(kubernetes.daemon_set.current_scheduled): The number of nodes that are
// running at least 1 daemon pod and are supposed to run the daemon pod

// GAUGE(kubernetes.daemon_set.desired_scheduled): The total number of nodes
// that should be running the daemon pod (including nodes currently running the
// daemon pod)

// GAUGE(kubernetes.daemon_set.misscheduled): The number of nodes that are
// running the daemon pod, but are not supposed to run the daemon pod

// GAUGE(kubernetes.daemon_set.ready): The number of nodes that should be
// running the daemon pod and have one or more of the daemon pod running and
// ready

func datapointsForDaemonSet(ds *v1beta1.DaemonSet) []*datapoint.Datapoint {
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": ds.Namespace,
		"uid":                  string(ds.UID),
		"kubernetes_name":      ds.Name,
	}

	return []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.daemon_set.current_scheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.CurrentNumberScheduled)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			"kubernetes.daemon_set.desired_scheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.DesiredNumberScheduled)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			"kubernetes.daemon_set.misscheduled",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.NumberMisscheduled)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			"kubernetes.daemon_set.ready",
			dimensions,
			datapoint.NewIntValue(int64(ds.Status.NumberReady)),
			datapoint.Gauge,
			time.Now()),
	}
}
