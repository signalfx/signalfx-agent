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

var logger = log.WithFields(log.Fields{
	"monitorType": "kubernetes-cluster",
})

// DatapointCache holds an up to date copy of datapoints pertaining to the
// cluster.  It is updated whenever the HandleAdd method is called with new
// K8s resources.
type DatapointCache struct {
    sync.Mutex
	dpCache      map[cachedResourceKey][]*datapoint.Datapoint
	dimPropCache map[cachedResourceKey]*atypes.DimProperties
	useNodeName  bool
}

// NewDatapointCache creates a new clean cache
func NewDatapointCache(useNodeName bool) *DatapointCache {
	return &DatapointCache{
		dpCache:      make(map[cachedResourceKey][]*datapoint.Datapoint),
		dimPropCache: make(map[cachedResourceKey]*atypes.DimProperties),
		useNodeName:  useNodeName,
	}
}

func keyForObject(obj runtime.Object) (cachedResourceKey, error) {
	kind := obj.GetObjectKind().GroupVersionKind()
    log.Info(obj.GetObjectKind())
	oma, ok := obj.(metav1.ObjectMetaAccessor)

	if !ok || oma.GetObjectMeta() == nil {
		return cachedResourceKey{}, errors.New("K8s object is not of the expected form")
	}
	return cachedResourceKey{
		Kind: kind,
		UID:  oma.GetObjectMeta().GetUID(),
	}, nil
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
	delete(dc.dimPropCache, key)

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
        dimProps = dimPropsForReplicaSet(o)
	case *v1.ResourceQuota:
		dps = datapointsForResourceQuota(o)
	case *v1.Node:
		dps = datapointsForNode(o, dc.useNodeName)
		dimProps = dimPropsForNode(o, dc.useNodeName)
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
	if dimProps != nil {
        dc.addDimPropsToCache(key, dimProps)
	}

	return key
}

type propertyLink struct {
    SourceProperty string
    SourceKind     string
    SourceJoinKey  string
    TargetProperty string
    TargetKind     string
    TargetJoinKey  string
}

func (dc *DatapointCache) addDimPropsToCache(key cachedResourceKey, dimProps *atypes.DimProperties) {
    links := []propertyLink{
        propertyLink{
            SourceKind:     "ReplicaSet",
            SourceProperty: "deployment",
            SourceJoinKey:  "name",
            TargetKind:     "Pod",
            TargetProperty: "deployment",
            TargetJoinKey:  "replicaSet",
        },
    }

    for _, link := range links {
       // If the dim props are for a pod, go through all the existing replica set
       // dim props and find one that has key.Kind.Kind == "ReplicaSet" and
       // key.UID == dimProps["replicaSetUID"] and then pull the "deployment"
       // properties off the replica set dim props and put it on the current pod
       // dimProps and put it in the cache
        if key.Kind.Kind == link.TargetKind {
            for k := range dc.dimPropCache {
                if k.Kind.Kind == link.SourceKind {
                    props := dc.dimPropCache[k].Properties
                    log.Info("Found Pod with matching replicaSet")
                    if props[link.SourceJoinKey] == dimProps.Properties[link.TargetJoinKey] {
                        log.WithFields(log.Fields{
                            "deployment": props[link.SourceProperty],
                            "replicaSet": props["name"],
                            "pod_uid": key,
                        }).Info("new pod: syncing deployments to pods via replicaSet")
                        dimProps.Properties[link.TargetProperty] = props[link.SourceProperty]
                    }
                }
            }
        }

       // If the dim props are for a replica set, go through all of the existing
       // pod dim props and do something similar to propagate the deployment to
       // the pods
        if key.Kind.Kind == link.SourceKind {
            for k := range dc.dimPropCache {
                if k.Kind.Kind == link.TargetKind {
                    props := dc.dimPropCache[k].Properties
                    if props[link.TargetJoinKey] == dimProps.Properties[link.SourceJoinKey] {
                        log.WithFields(log.Fields{
                            "deployment": props[link.SourceProperty],
                            "replicaSet": props["name"],
                            "pod_uid": k,
                        }).Info("new replicaSet: syncing deployment to pods with this replicaSet")
                        props[link.TargetProperty] = dimProps.Properties[link.SourceProperty]
                    }
                }
            }
        }
    }

    dc.dimPropCache[key] = dimProps
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

// AllDimProperties returns any dimension properties pertaining to the cluster
func (dc *DatapointCache) AllDimProperties() []*atypes.DimProperties {
	dimProps := make([]*atypes.DimProperties, 0)

	dc.Lock()
	defer dc.Unlock()

	for k := range dc.dimPropCache {
		if dc.dimPropCache[k] != nil {
			dimProps = append(dimProps, dc.dimPropCache[k])
		}
	}

	return dimProps
}
