package metrics

import (
	"errors"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// ContainerID is some type of unique id for containers
type ContainerID string

type cachedResourceKey struct {
	Kind schema.GroupVersionKind
	UID  types.UID
}

// DatapointCache holds an up to date copy of datapoints pertaining to the
// cluster.  It is updated whenever the HandleChange method is called with new
// K8s resources.
type DatapointCache struct {
	dpCache      map[cachedResourceKey][]*datapoint.Datapoint
	dimPropCache map[cachedResourceKey]*atypes.DimProperties
	lastNodes    map[types.UID]*v1.Node
	mutex        sync.Mutex
}

// NewDatapointCache creates a new clean cache
func NewDatapointCache() *DatapointCache {
	return &DatapointCache{
		dpCache:      make(map[cachedResourceKey][]*datapoint.Datapoint),
		dimPropCache: make(map[cachedResourceKey]*atypes.DimProperties),
		lastNodes:    make(map[types.UID]*v1.Node),
	}
}

func keyForObject(obj runtime.Object) (cachedResourceKey, error) {
	kind := obj.GetObjectKind().GroupVersionKind()
	oma, ok := obj.(metav1.ObjectMetaAccessor)

	if !ok || oma.GetObjectMeta() == nil {
		return cachedResourceKey{}, errors.New("K8s object is not of the expected form")
	}

	return cachedResourceKey{
		Kind: kind,
		UID:  oma.GetObjectMeta().GetUID(),
	}, nil
}

// Lock allows users of the cache to lock it when doing complex operations
func (dc *DatapointCache) Lock() {
	dc.mutex.Lock()
}

// Unlock allows users of the cache to unlock it after doing complex operations
func (dc *DatapointCache) Unlock() {
	dc.mutex.Unlock()
}

// DeleteByKey delete a cache entry by key.  The supplied interface MUST be the
// same type returned by Handle[Add|Delete].  MUST HOLD LOCK!
func (dc *DatapointCache) DeleteByKey(key interface{}) {
	delete(dc.dpCache, key.(cachedResourceKey))
	delete(dc.dimPropCache, key.(cachedResourceKey))
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

	delete(dc.dpCache, key)
	if o, ok := oldObj.(*v1.Node); ok {
		delete(dc.lastNodes, o.UID)
	}
	return key
}

// HandleAdd accepts a new (or updated) object and updates the datapoint/prop
// cache as needed.  MUST HOLD LOCK!!
func (dc *DatapointCache) HandleAdd(newObj runtime.Object) interface{} {
	var dps []*datapoint.Datapoint
	var dimProps *atypes.DimProperties

	switch o := newObj.(type) {
	case *v1.Pod:
		dps = datapointsForPod(o)
		dimProps = dimPropsForPod(o)
	case *v1.Namespace:
		dps = datapointsForNamespace(o)
	case *v1.ReplicationController:
		dps = datapointsForReplicationController(o)
	case *v1beta1.DaemonSet:
		dps = datapointsForDaemonSet(o)
	case *v1beta1.Deployment:
		dps = datapointsForDeployment(o)
	case *v1beta1.ReplicaSet:
		dps = datapointsForReplicaSet(o)
	case *v1.Node:
		// Nodes update incredibly often so don't do all the work figuring out
		// datapoints if they haven't changed in a way we care about.
		if !nodesDifferent(o, dc.lastNodes[o.UID]) {
			return nil
		}

		dps = datapointsForNode(o)
		dimProps = dimPropsForNode(o)
		dc.lastNodes[o.UID] = o
	default:
		log.WithFields(log.Fields{
			"obj": spew.Sdump(newObj),
		}).Error("Unknown object type in HandleChange")
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
	if dimProps != nil {
		dc.dimPropCache[key] = dimProps
	}

	return key
}

// AllDatapoints returns all of the cached datapoints.
func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dps := make([]*datapoint.Datapoint, 0)

	dc.mutex.Lock()
	defer dc.mutex.Unlock()

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

// AllDimProperties returns any dimension properties pertaining to the cluster
func (dc *DatapointCache) AllDimProperties() []*atypes.DimProperties {
	dimProps := make([]*atypes.DimProperties, 0)

	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	for k := range dc.dimPropCache {
		if dc.dimPropCache[k] != nil {
			dimProps = append(dimProps, dc.dimPropCache[k])
		}
	}

	return dimProps
}
