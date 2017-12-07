package metrics

import (
	"sync"

	"github.com/signalfx/golib/datapoint"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
)

type resourceMetrics interface {
	Datapoints() []*datapoint.Datapoint
	Add(runtime.Object)
	Remove(runtime.Object)
}

// ContainerID is some type of unique id for containers
type ContainerID string

// DatapointCache maintains an up to date set of datapoints at all times and
// just send them on demand. All count metrics (e.g. pod/container/deployment
// counts) should be derived from other metrics to cut down on DPM.
type DatapointCache struct {
	podMetrics                   resourceMetrics
	daemonSetMetrics             resourceMetrics
	deploymentMetrics            resourceMetrics
	replicationControllerMetrics resourceMetrics
	replicaSetMetrics            resourceMetrics
	nodeMetrics                  resourceMetrics

	Mutex sync.Mutex
	// Used to optimize slice creation when collecting all datapoints
	roughDatapointCount int
}

// NewDatapointCache creates an empty datapoint cache
func NewDatapointCache() *DatapointCache {
	return &DatapointCache{
		podMetrics:                   newPodMetrics(),
		daemonSetMetrics:             newDaemonSetMetrics(),
		deploymentMetrics:            newDeploymentMetrics(),
		replicationControllerMetrics: newReplicationControllerMetrics(),
		replicaSetMetrics:            newReplicaSetMetrics(),
		nodeMetrics:                  newNodeMetrics(),
	}
}

// AllDatapoints returns all of the datapoints as a slice.  Mutex must be held
// throughout use of the returned datapoint slice unless you make a copy of all
// datapoints!!
// TODO: figure out how to make this more automatic and less verbose
func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	dps = append(dps, dc.podMetrics.Datapoints()...)
	dps = append(dps, dc.daemonSetMetrics.Datapoints()...)
	dps = append(dps, dc.deploymentMetrics.Datapoints()...)
	dps = append(dps, dc.replicationControllerMetrics.Datapoints()...)
	dps = append(dps, dc.replicaSetMetrics.Datapoints()...)
	dps = append(dps, dc.nodeMetrics.Datapoints()...)

	return dps
}

// HandleChange updates the datapoint cache when called with the old and new
// resources.  Delete only includes the old, and adds only includes the new.
func (dc *DatapointCache) HandleChange(oldObj, newObj runtime.Object) {
	dc.Mutex.Lock()
	defer dc.Mutex.Unlock()

	var handler resourceMetrics

	var objToTest runtime.Object
	if oldObj != nil {
		objToTest = oldObj
	} else {
		objToTest = newObj
	}

	switch objToTest.(type) {
	case *v1.Pod:
		handler = dc.podMetrics
	case *v1.ReplicationController:
		handler = dc.replicationControllerMetrics
	case *v1beta1.DaemonSet:
		handler = dc.daemonSetMetrics
	case *v1beta1.Deployment:
		handler = dc.deploymentMetrics
	case *v1beta1.ReplicaSet:
		handler = dc.replicaSetMetrics
	case *v1.Node:
		handler = dc.nodeMetrics
	default:
		log.WithFields(log.Fields{
			"objectType": objToTest,
		}).Error("Unknown object type in HandleChange")
		return
	}

	// Updates of existing objects cause old DPs to be removed and re-added
	if oldObj != nil {
		handler.Remove(oldObj)
	}
	if newObj != nil {
		handler.Add(newObj)
	}
}
