package metrics

import (
	"errors"
	"regexp"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	atypes "github.com/signalfx/neo-agent/monitors/types"
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

type CachedResourceKey struct {
	Kind schema.GroupVersionKind
	UID  types.UID
}

var propNameSanitizer = regexp.MustCompile(`[./]`)

type DatapointCache struct {
	dpCache      map[CachedResourceKey][]*datapoint.Datapoint
	dimPropCache map[CachedResourceKey]*atypes.DimProperties
	mutex        sync.Mutex
}

func NewDatapointCache() *DatapointCache {
	return &DatapointCache{
		dpCache:      make(map[CachedResourceKey][]*datapoint.Datapoint),
		dimPropCache: make(map[CachedResourceKey]*atypes.DimProperties),
	}
}

func keyForObject(obj runtime.Object) (*CachedResourceKey, error) {
	kind := obj.GetObjectKind().GroupVersionKind()
	oma, ok := obj.(metav1.ObjectMetaAccessor)

	if !ok || oma.GetObjectMeta() == nil {
		return nil, errors.New("K8s object is not of the expected form")
	}

	return &CachedResourceKey{
		Kind: kind,
		UID:  oma.GetObjectMeta().GetUID(),
	}, nil
}

// HandleChange updates the datapoint cache when called with the old and new
// resources.  Delete only includes the old, and adds only includes the new.
func (dc *DatapointCache) HandleChange(oldObj, newObj runtime.Object) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if oldObj != nil {
		key, err := keyForObject(oldObj)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"obj":   spew.Sdump(oldObj),
			}).Error("Could not get cache key")
			return
		}

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
	case *v1.ReplicationController:
		dps = datapointsForReplicationController(o)
	case *v1beta1.DaemonSet:
		dps = datapointsForDaemonSet(o)
	case *v1beta1.Deployment:
		dps = datapointsForDeployment(o)
	case *v1beta1.ReplicaSet:
		dps = datapointsForReplicaSet(o)
	case *v1.Node:
		dps = datapointsForNode(o)
		dimProps = dimPropsForNode(o)
	default:
		log.WithFields(log.Fields{
			"obj": spew.Sdump(newObj),
		}).Error("Unknown object type in HandleChange")
		return
	}

	dc.dpCache[*key] = dps
	dc.dimPropCache[*key] = dimProps
}

func (dc *DatapointCache) AllDatapoints() []*datapoint.Datapoint {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	dps := make([]*datapoint.Datapoint, 0)

	for k := range dc.dpCache {
		if dc.dpCache[k] != nil {
			dps = append(dps, dc.dpCache[k]...)
		}
	}

	return dps
}

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
