package kubernetes

import (
	"fmt"
	"log"
	"sync"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/utils"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/types"
)

// ContainerID is some type of unique id for containers
type ContainerID string

// ReplicaDPs hold datapoints for replication resources.  There are other
// metrics we could pull that give a more fine-grained view into non-available
// replicas, but these two should suffice for most users needs to start.
type ReplicaDPs struct {
	DesiredReplicas   *datapoint.Datapoint
	AvailableReplicas *datapoint.Datapoint
}

// DaemonSetDPs hold datapoints relevant to Daemon Sets
type DaemonSetDPs struct {
	CurrentNumberScheduled *datapoint.Datapoint
	DesiredNumberScheduled *datapoint.Datapoint
	NumberMisscheduled     *datapoint.Datapoint
	NumberReady            *datapoint.Datapoint
}

// DatapointCache maintains an up to date set of datapoints at all times and
// just send them on demand. All count metrics (e.g. pod/container/deployment
// counts) should be derived from other metrics to cut down on DPM.
type DatapointCache struct {
	// Int value of the datapoint.Datapoint corresponds to the phase (see `phaseToInt` below)
	PodPhases map[types.UID]*datapoint.Datapoint
	// These will cap at 5 according to
	// https://kubernetes.io/docs/api-reference/v1.5/#containerstatus-v1
	ContainerRestartCount  map[ContainerID]*datapoint.Datapoint
	DaemonSets             map[types.UID]DaemonSetDPs
	Deployments            map[types.UID]ReplicaDPs
	ReplicationControllers map[types.UID]ReplicaDPs
	ReplicaSets            map[types.UID]ReplicaDPs

	Mutex sync.Mutex
	// Used to optimize slice creation when collecting all datapoints
	roughDatapointCount int
}

func newDatapointCache() *DatapointCache {
	return &DatapointCache{
		PodPhases:              make(map[types.UID]*datapoint.Datapoint),
		ContainerRestartCount:  make(map[ContainerID]*datapoint.Datapoint),
		DaemonSets:             make(map[types.UID]DaemonSetDPs),
		Deployments:            make(map[types.UID]ReplicaDPs),
		ReplicationControllers: make(map[types.UID]ReplicaDPs),
		ReplicaSets:            make(map[types.UID]ReplicaDPs),
		// Just pick something non-zero to start
		roughDatapointCount: 100,
	}
}

// AllDatapoints returns all of the datapoints as a slice.  Mutex must be held
// throughout use of the returned datapoint slice unless you make a copy of all
// datapoints!!
// TODO: figure out how to make this more automatic and less verbose
func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0, dc.roughDatapointCount)

	for _, dp := range dc.PodPhases {
		dps = append(dps, dp)
	}

	for _, dp := range dc.ContainerRestartCount {
		dps = append(dps, dp)
	}

	for _, dsdps := range dc.DaemonSets {
		dps = append(dps, []*datapoint.Datapoint{
			dsdps.CurrentNumberScheduled,
			dsdps.DesiredNumberScheduled,
			dsdps.NumberMisscheduled,
			dsdps.NumberReady,
		}...)
	}

	replicaMaps := []map[types.UID]ReplicaDPs{dc.Deployments, dc.ReplicationControllers, dc.ReplicaSets}
	for _, reps := range replicaMaps {
		for _, rdps := range reps {
			dps = append(dps, []*datapoint.Datapoint{
				rdps.DesiredReplicas,
				rdps.AvailableReplicas,
			}...)
		}
	}

	dc.roughDatapointCount = len(dps)

	return dps
}

// HandleChange updates the datapoint cache when called with the old and new
// resources.  Delete only includes the old, and adds only includes the new.
func (dc *DatapointCache) HandleChange(oldObj, newObj runtime.Object) {
	dc.Mutex.Lock()
	defer dc.Mutex.Unlock()

	var removeFunc func(runtime.Object)
	var addFunc func(runtime.Object)

	var objToTest runtime.Object
	if oldObj != nil {
		objToTest = oldObj
	} else {
		objToTest = newObj
	}

	switch objToTest.(type) {
	case *v1.Pod:
		removeFunc = dc.removePodDps
		addFunc = dc.addPodDps
	case *v1.ReplicationController:
		removeFunc = dc.removeRcDps
		addFunc = dc.addRcDps
	case *v1beta1.DaemonSet:
		removeFunc = dc.removeDaemonSetDps
		addFunc = dc.addDaemonSetDps
	case *v1beta1.Deployment:
		removeFunc = dc.removeDeploymentDps
		addFunc = dc.addDeploymentDps
	case *v1beta1.ReplicaSet:
		removeFunc = dc.removeReplicaSetDps
		addFunc = dc.addReplicaSetDps
	default:
		log.Printf("Unknown object type in HandleChange: %#v", objToTest)
		return
	}

	// Updates of existing objects cause old DPs to be removed and re-added
	if oldObj != nil {
		removeFunc(oldObj)
	}
	if newObj != nil {
		addFunc(newObj)
	}
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

// Assumes mutex is held by caller
func (dc *DatapointCache) addPodDps(obj runtime.Object) {
	pod := obj.(*v1.Pod)
	dimensions := map[string]string{
		"metric_source": "kubernetes",
		// Try and be consistent with other plugin dimensions, despite
		// verbosity
		"kubernetes_pod_namespace": pod.Namespace,
		"pod_uid":                  string(pod.UID),
		"kubernetes_pod_name":      pod.Name,
		"kubernetes_node":          pod.Spec.NodeName,
	}
	for name, value := range pod.Labels {
		dimensions["label_"+name] = value
	}

	dc.PodPhases[pod.UID] = &datapoint.Datapoint{
		Metric:     "kubernetes.pod_phase",
		Dimensions: dimensions,
		Value:      datapoint.NewIntValue(phaseToInt(pod.Status.Phase)),
		MetricType: datapoint.Gauge,
	}

	for _, cs := range pod.Status.ContainerStatuses {
		contDims := utils.CloneStringMap(dimensions)
		contDims["container_name"] = cs.Name
		contDims["container_image"] = cs.Image
		dc.ContainerRestartCount[makeContUID(pod.UID, cs.Name)] = &datapoint.Datapoint{
			Metric:     "kubernetes.container_restart_count",
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(cs.RestartCount)),
			MetricType: datapoint.Gauge,
		}
	}
}

func (dc *DatapointCache) removePodDps(obj runtime.Object) {
	pod := obj.(*v1.Pod)
	delete(dc.PodPhases, pod.UID)
	for _, cs := range pod.Status.ContainerStatuses {
		delete(dc.ContainerRestartCount, makeContUID(pod.UID, cs.Name))
	}
}

func (dc *DatapointCache) addDaemonSetDps(obj runtime.Object) {
	ds := obj.(*v1beta1.DaemonSet)

	dimensions := map[string]string{
		"metric_source":            "kubernetes",
		"kubernetes_pod_namespace": ds.Namespace,
		"uid":             string(ds.UID),
		"kubernetes_name": ds.Name,
	}
	dc.DaemonSets[ds.UID] = DaemonSetDPs{
		CurrentNumberScheduled: &datapoint.Datapoint{
			Metric:     "kubernetes.daemon_set.current_scheduled",
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(ds.Status.CurrentNumberScheduled)),
			MetricType: datapoint.Gauge,
		},
		DesiredNumberScheduled: &datapoint.Datapoint{
			Metric:     "kubernetes.daemon_set.desired_scheduled",
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(ds.Status.DesiredNumberScheduled)),
			MetricType: datapoint.Gauge,
		},
		NumberMisscheduled: &datapoint.Datapoint{
			Metric:     "kubernetes.daemon_set.misscheduled",
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(ds.Status.NumberMisscheduled)),
			MetricType: datapoint.Gauge,
		},
		NumberReady: &datapoint.Datapoint{
			Metric:     "kubernetes.daemon_set.ready",
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(ds.Status.NumberReady)),
			MetricType: datapoint.Gauge,
		},
	}
}
func (dc *DatapointCache) removeDaemonSetDps(obj runtime.Object) {
	ds := obj.(*v1beta1.DaemonSet)
	delete(dc.DaemonSets, ds.UID)
}

func makeReplicaDPs(resource string, dimensions map[string]string, desired, available int32) ReplicaDPs {
	return ReplicaDPs{
		DesiredReplicas: &datapoint.Datapoint{
			Metric:     fmt.Sprintf("kubernetes.%s.desired", resource),
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(desired)),
			MetricType: datapoint.Gauge,
		},
		AvailableReplicas: &datapoint.Datapoint{
			Metric:     fmt.Sprintf("kubernetes.%s.available", resource),
			Dimensions: dimensions,
			Value:      datapoint.NewIntValue(int64(available)),
			MetricType: datapoint.Gauge,
		},
	}
}

func (dc *DatapointCache) addRcDps(obj runtime.Object) {
	rc := obj.(*v1.ReplicationController)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rc.Namespace,
		"uid":             string(rc.UID),
		"kubernetes_name": rc.Name,
	}

	if rc.Spec.Replicas == nil {
		return
	}
	dc.ReplicationControllers[rc.UID] = makeReplicaDPs("replication_controller", dimensions,
		*rc.Spec.Replicas, rc.Status.AvailableReplicas)
}

func (dc *DatapointCache) removeRcDps(obj runtime.Object) {
	rc := obj.(*v1.ReplicationController)
	delete(dc.ReplicationControllers, rc.UID)
}

func (dc *DatapointCache) addDeploymentDps(obj runtime.Object) {
	dep := obj.(*v1beta1.Deployment)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": dep.Namespace,
		"uid":             string(dep.UID),
		"kubernetes_name": dep.Name,
	}

	if dep.Spec.Replicas == nil { // || dep.Status.AvailableReplicas == nil {
		return
	}
	dc.Deployments[dep.UID] = makeReplicaDPs("deployment", dimensions,
		*dep.Spec.Replicas, dep.Status.AvailableReplicas)
}

func (dc *DatapointCache) removeDeploymentDps(obj runtime.Object) {
	d := obj.(*v1beta1.Deployment)
	delete(dc.Deployments, d.UID)
}

func (dc *DatapointCache) addReplicaSetDps(obj runtime.Object) {
	rs := obj.(*v1beta1.ReplicaSet)
	dimensions := map[string]string{
		"metric_source":        "kubernetes",
		"kubernetes_namespace": rs.Namespace,
		"uid":             string(rs.UID),
		"kubernetes_name": rs.Name,
	}

	if rs.Spec.Replicas == nil { //|| rs.Status.AvailableReplicas == nil {
		return
	}
	dc.ReplicaSets[rs.UID] = makeReplicaDPs("replica_set", dimensions,
		*rs.Spec.Replicas, rs.Status.AvailableReplicas)
}

func (dc *DatapointCache) removeReplicaSetDps(obj runtime.Object) {
	rs := obj.(*v1beta1.ReplicaSet)
	delete(dc.ReplicaSets, rs.UID)
}
