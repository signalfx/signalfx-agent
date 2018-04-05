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
	mutex        sync.Mutex
}

// NewDatapointCache creates a new clean cache
func NewDatapointCache() *DatapointCache {
	return &DatapointCache{
		dpCache:      make(map[cachedResourceKey][]*datapoint.Datapoint),
		dimPropCache: make(map[cachedResourceKey]*atypes.DimProperties),
	}
}

func keyForObject(obj runtime.Object) (*cachedResourceKey, error) {
	kind := obj.GetObjectKind().GroupVersionKind()
	oma, ok := obj.(metav1.ObjectMetaAccessor)

	if !ok || oma.GetObjectMeta() == nil {
		return nil, errors.New("K8s object is not of the expected form")
	}

	return &cachedResourceKey{
		Kind: kind,
		UID:  oma.GetObjectMeta().GetUID(),
	}, nil
}

// HandleChange updates the datapoint cache when called with the old and new
// resources.  Delete only includes the old, and adds only includes the new.
func (dc *DatapointCache) HandleChange(oldObj, newObj runtime.Object) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	var prevCached []*datapoint.Datapoint
	if oldObj != nil {
		key, err := keyForObject(oldObj)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"obj":   spew.Sdump(oldObj),
			}).Error("Could not get cache key")
			return
		}

		prevCached = dc.dpCache[*key]
		delete(dc.dpCache, *key)
	}

	if newObj == nil {
		return
	}

	key, err := keyForObject(newObj)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"obj":   spew.Sdump(newObj),
		}).Error("Could not get cache key")
	}

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
		if oldObj == nil || nodesDifferent(o, oldObj.(*v1.Node)) {
			dps = datapointsForNode(o)
			dimProps = dimPropsForNode(o)
		} else if prevCached != nil {
			// Reinsert it into the cache since we deleted it above since
			// oldObj != nil but avoid recalculating dps to avoid excess CPU.
			dc.dpCache[*key] = prevCached
		}
	default:
		log.WithFields(log.Fields{
			"obj": spew.Sdump(newObj),
		}).Error("Unknown object type in HandleChange")
		return
	}

	if dps != nil {
		dc.dpCache[*key] = dps
	}
	if dimProps != nil {
		dc.dimPropCache[*key] = dimProps
	}
}

// AllDatapoints returns all of the cached datapoints.
func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dps := make([]*datapoint.Datapoint, 0)

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
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dimProps := make([]*atypes.DimProperties, 0)

	for k := range dc.dimPropCache {
		if dc.dimPropCache[k] != nil {
			dimProps = append(dimProps, dc.dimPropCache[k])
		}
	}

	return dimProps
}
