package metrics

import (
	"fmt"
	"time"

	"github.com/signalfx/golib/datapoint"
)

// replicaDPs hold datapoints for replication resources.  There are other
// metrics we could pull that give a more fine-grained view into non-available
// replicas, but these two should suffice for most users needs to start.
type replicaDPs struct {
	DesiredReplicas   *datapoint.Datapoint
	AvailableReplicas *datapoint.Datapoint
}

func makeReplicaDPs(resource string, dimensions map[string]string, desired, available int32) replicaDPs {
	return replicaDPs{
		DesiredReplicas: datapoint.New(
			fmt.Sprintf("kubernetes.%s.desired", resource),
			dimensions,
			datapoint.NewIntValue(int64(desired)),
			datapoint.Gauge,
			time.Now()),
		AvailableReplicas: datapoint.New(
			fmt.Sprintf("kubernetes.%s.available", resource),
			dimensions,
			datapoint.NewIntValue(int64(available)),
			datapoint.Gauge,
			time.Now()),
	}
}

func (rep *replicaDPs) Datapoints() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		rep.DesiredReplicas,
		rep.AvailableReplicas,
	}
}
