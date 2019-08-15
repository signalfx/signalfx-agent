package metrics

import (
	"errors"
	"sync"

	"k8s.io/api/autoscaling/v2beta1"

	"github.com/davecgh/go-spew/spew"
	quota "github.com/openshift/api/quota/v1"
	"github.com/signalfx/golib/datapoint"
	k8sutil "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/utils"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ContainerID is some type of unique id for containers
type ContainerID string

var logger = log.WithFields(log.Fields{
	"monitorType": "kubernetes-cluster",
})

// DatapointCache holds an up to date copy of datapoints pertaining to the
// cluster.  It is updated whenever the HandleAdd method is called with new
// K8s resources.
type DatapointCache struct {
	sync.Mutex
	dpCache                    map[types.UID][]*datapoint.Datapoint
	dimCache                   map[types.UID]*atypes.Dimension
	uidKindCache               map[types.UID]string
	podCache                   *k8sutil.PodCache
	serviceCache               *k8sutil.ServiceCache
	replicaSetCache            *k8sutil.ReplicaSetCache
	jobCache                   *k8sutil.JobCache
	useNodeName                bool
	nodeConditionTypesToReport []string
}

// NewDatapointCache creates a new clean cache
func NewDatapointCache(useNodeName bool, nodeConditionTypesToReport []string) *DatapointCache {
	return &DatapointCache{
		dpCache:                    make(map[types.UID][]*datapoint.Datapoint),
		dimCache:                   make(map[types.UID]*atypes.Dimension),
		uidKindCache:               make(map[types.UID]string),
		podCache:                   k8sutil.NewPodCache(),
		serviceCache:               k8sutil.NewServiceCache(),
		replicaSetCache:            k8sutil.NewReplicaSetCache(),
		jobCache:                   k8sutil.NewJobCache(),
		useNodeName:                useNodeName,
		nodeConditionTypesToReport: nodeConditionTypesToReport,
	}
}

func keyForObject(obj runtime.Object) (types.UID, error) {
	var key types.UID
	oma, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok || oma.GetObjectMeta() == nil {
		return key, errors.New("kubernetes object is not of the expected form")
	}
	key = oma.GetObjectMeta().GetUID()
	return key, nil
}

// DeleteByKey delete a cache entry by key.  The supplied interface MUST be the
// same type returned by Handle[Add|Delete].  MUST HOLD LOCK!
func (dc *DatapointCache) DeleteByKey(key interface{}) {
	cacheKey := key.(types.UID)
	var err error
	switch dc.uidKindCache[cacheKey] {
	case "Pod":
		err = dc.podCache.DeleteByKey(cacheKey)
	case "Service":
		err = dc.handleDeleteService(cacheKey)
	case "ReplicaSet":
		err = dc.replicaSetCache.DeleteByKey(cacheKey)
	case "Job":
		err = dc.jobCache.DeleteByKey(cacheKey)
	}
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"UID":   cacheKey,
		}).Error("an error occurred while deleting a key from an internal cache")
	}
	delete(dc.uidKindCache, cacheKey)
	delete(dc.dpCache, cacheKey)
	delete(dc.dimCache, cacheKey)
}

// HandleDelete accepts an object that has been deleted and removes the
// associated datapoints/props from the cache.  MUST HOLD LOCK!!
func (dc *DatapointCache) HandleDelete(oldObj runtime.Object) interface{} {
	key, err := keyForObject(oldObj)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"obj":   spew.Sdump(oldObj),
		}).Error("Could not get cache key")
		return nil
	}
	dc.DeleteByKey(key)
	return key
}

// HandleAdd accepts a new (or updated) object and updates the datapoint/prop
// cache as needed.  MUST HOLD LOCK!!
// nolint: funlen
func (dc *DatapointCache) HandleAdd(newObj runtime.Object) interface{} {
	var dps []*datapoint.Datapoint
	var dim *atypes.Dimension
	var kind string

	switch o := newObj.(type) {
	case *v1.Pod:
		dps, dim = dc.handleAddPod(o)
		kind = "Pod"
	case *v1.Namespace:
		dps = datapointsForNamespace(o)
		kind = "Namespace"
	case *v1.ReplicationController:
		dps = datapointsForReplicationController(o)
		kind = "ReplicationController"
	case *appsv1.DaemonSet:
		dps = datapointsForDaemonSet(o)
		dim = dimensionForDaemonSet(o)
		kind = "DaemonSet"
	case *appsv1.Deployment:
		dps = datapointsForDeployment(o)
		dim = dimensionForDeployment(o)
		kind = "Deployment"
	case *appsv1.ReplicaSet:
		dps, dim = dc.handleAddReplicaSet(o)
		kind = "ReplicaSet"
	case *v1.ResourceQuota:
		dps = datapointsForResourceQuota(o)
		kind = "ResourceQuota"
	case *v1.Node:
		dps = datapointsForNode(o, dc.useNodeName, dc.nodeConditionTypesToReport)
		dim = dimensionForNode(o, dc.useNodeName)
		kind = "Node"
	case *v1.Service:
		dc.handleAddService(o)
		kind = "Service"
	case *appsv1.StatefulSet:
		dps = datapointsForStatefulSet(o)
		dim = dimensionForStatefulSet(o)
		kind = "StatefulSet"
	case *quota.ClusterResourceQuota:
		dps = datapointsForClusterQuotas(o)
		kind = "ClusterResourceQuota"
	case *batchv1.Job:
		dps, dim = dc.handleAddJob(o)
		kind = "Job"
	case *batchv1beta1.CronJob:
		dps = datapointsForCronJob(o)
		dim = dimensionForCronJob(o)
		kind = "CronJob"
	case *v2beta1.HorizontalPodAutoscaler:
		dps = datapointsForHpa(o)
		dim = dimensionForHpa(o)
		kind = "HorizontalPodAutoscaler"
	default:
		log.WithFields(log.Fields{
			"obj": spew.Sdump(newObj),
		}).Error("Unknown object type in HandleAdd")
		return nil
	}

	key, err := keyForObject(newObj)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"obj":   spew.Sdump(newObj),
		}).Error("Could not get cache key")
		return nil
	}

	if dps != nil {
		dc.dpCache[key] = dps
	}
	if kind != "" {
		dc.uidKindCache[key] = kind
	}
	if dim != nil {
		dc.addDimToCache(key, dim)
	}

	return key
}

// addDimToCache maps and syncs properties from different resources together and adds
// them to the cache
func (dc *DatapointCache) addDimToCache(key types.UID, dim *atypes.Dimension) {
	dc.dimCache[key] = dim
}

// handleAddPod adds a pod to the internal pod cache and gets the
// datapoints and dim for the pod.
func (dc *DatapointCache) handleAddPod(pod *v1.Pod) ([]*datapoint.Datapoint,
	*atypes.Dimension) {
	if !dc.podCache.IsCached(pod) {
		dc.podCache.AddPod(pod)
	}
	cachedPod := dc.podCache.GetCachedPod(pod.UID)
	dps := datapointsForPod(pod)

	if cachedPod != nil {
		dim := dimensionForPod(cachedPod, dc.serviceCache, dc.replicaSetCache, dc.jobCache)
		return dps, dim
	}
	return dps, nil
}

// handleAddService adds a service to internal cache and, if needed,
// will check cached pods if the service matches to add service name
// and service UID properties to the pod.
func (dc *DatapointCache) handleAddService(svc *v1.Service) {
	if !dc.serviceCache.IsCached(svc) {
		serviceAdded := dc.serviceCache.AddService(svc)
		if serviceAdded {
			for _, podUID := range dc.podCache.GetPodsInNamespace(svc.Namespace) {
				cachedPod := dc.podCache.GetCachedPod(podUID)
				if cachedPod != nil {
					dim := dimensionForPod(cachedPod, dc.serviceCache, dc.replicaSetCache, dc.jobCache)
					if dim != nil {
						dc.addDimToCache(podUID, dim)
					}
				}
			}
		}
	}
}

// handleDeleteService will remove a service from the internal cache
// and remove the service tags on it's matching pods.
func (dc *DatapointCache) handleDeleteService(svcUID types.UID) error {
	cachedPods, err := dc.serviceCache.DeleteByKey(svcUID)
	if err != nil {
		return err
	}
	for _, podUID := range cachedPods {
		cachedPod := dc.podCache.GetCachedPod(podUID)
		if cachedPod != nil {
			dim := dimensionForPod(cachedPod, dc.serviceCache, dc.replicaSetCache, dc.jobCache)
			if dim != nil {
				dc.addDimToCache(podUID, dim)
			}
		}
	}
	return nil
}

// handleAddReplicaSet adds a replicaset to the internal cache and
// returns the datapoints and dim for the replicaset.
func (dc *DatapointCache) handleAddReplicaSet(rs *appsv1.ReplicaSet) ([]*datapoint.Datapoint, *atypes.Dimension) {
	if !dc.replicaSetCache.IsCached(rs) {
		dc.replicaSetCache.AddReplicaSet(rs)
		for _, podUID := range dc.podCache.GetPodsInNamespace(rs.Namespace) {
			cachedPod := dc.podCache.GetCachedPod(podUID)
			if cachedPod != nil {
				dim := dimensionForPod(cachedPod, dc.serviceCache, dc.replicaSetCache, dc.jobCache)
				if dim != nil {
					dc.addDimToCache(podUID, dim)
				}
			}
		}
	}
	dps := datapointsForReplicaSet(rs)
	dim := dimensionForReplicaSet(rs)
	return dps, dim
}

// handleAddJob adds a job to the internal cache and returns the datapoints and dim
// for the given job. Jobs should always be created before pods, but incase we receive
// the job out of sync, we can still check if the pod came in before the job.
// Potential optimization would be to not check this and always assume they come in order.
func (dc *DatapointCache) handleAddJob(job *batchv1.Job) ([]*datapoint.Datapoint, *atypes.Dimension) {
	if !dc.jobCache.IsCached(job) {
		dc.jobCache.AddJob(job)
		for _, podUID := range dc.podCache.GetPodsInNamespace(job.Namespace) {
			cachedPod := dc.podCache.GetCachedPod(podUID)
			if cachedPod != nil {
				dim := dimensionForPod(cachedPod, dc.serviceCache, dc.replicaSetCache, dc.jobCache)
				if dim != nil {
					dc.addDimToCache(podUID, dim)
				}
			}
		}
	}
	dps := datapointsForJob(job)
	dim := dimensionForJob(job)
	return dps, dim

}

// AllDatapoints returns all of the cached datapoints.
func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	dc.Lock()
	defer dc.Unlock()

	for k := range dc.dpCache {
		if dc.dpCache[k] != nil {
			for i := range dc.dpCache[k] {
				// Copy the datapoint since nothing in datapoints is thread
				// safe.
				dp := *dc.dpCache[k][i]
				dps = append(dps, &dp)
			}
		}
	}

	return dps
}

// AllDimensions returns any dimension properties pertaining to the cluster
func (dc *DatapointCache) AllDimensions() []*atypes.Dimension {
	dim := make([]*atypes.Dimension, 0)

	dc.Lock()
	defer dc.Unlock()

	for k := range dc.dimCache {
		if dc.dimCache[k] != nil {
			clonedDim := dc.dimCache[k].Copy()
			dim = append(dim, clonedDim)
		}
	}

	return dim
}
