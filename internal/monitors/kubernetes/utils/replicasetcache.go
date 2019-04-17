package utils

import (
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type replicasetSet map[types.UID]bool

// ReplicaSetCache is used for holding values we care about from a replicaset
// for quicker lookup than querying the API for them each time.
type ReplicaSetCache struct {
	namespaceRsUIDCache map[string]replicasetSet
	cachedReplicaSets   map[types.UID]*CachedReplicaSet
}

// NewReplicaSetCache creates a new replicaset cache
func NewReplicaSetCache() *ReplicaSetCache {
	return &ReplicaSetCache{
		namespaceRsUIDCache: make(map[string]replicasetSet),
		cachedReplicaSets:   make(map[types.UID]*CachedReplicaSet),
	}
}

// CachedReplicaSet is used for holding only the neccesarry fields we need
// for syncing deployment name and UID to pods
type CachedReplicaSet struct {
	UID           types.UID
	Name          string
	Namespace     string
	Deployment    *string
	DeploymentUID types.UID
}

func newCachedReplicaSet(rs *v1beta1.ReplicaSet) *CachedReplicaSet {
	var deployment *string
	var deploymentUID types.UID
	for _, or := range rs.OwnerReferences {
		if or.Kind == "Deployment" {
			deployment = &or.Name
			deploymentUID = or.UID
			break
		}
	}

	return &CachedReplicaSet{
		UID:           rs.UID,
		Name:          rs.Name,
		Namespace:     rs.Namespace,
		Deployment:    deployment,
		DeploymentUID: deploymentUID,
	}
}

// IsCached checks if a replicaset is already in the cache or if any of the
// the cached fields have changed.
func (rsc *ReplicaSetCache) IsCached(rs *v1beta1.ReplicaSet) bool {
	cachedRs, exists := rsc.cachedReplicaSets[rs.UID]

	return exists &&
		(cachedRs.Name == rs.Name) &&
		(cachedRs.Namespace == rs.Namespace)
}

// AddReplicaSet adds or updates a replicaset in the cache
// This function should only be called after rs.IsCached
// to prevent unnecessary updates to the internal cache.
func (rsc *ReplicaSetCache) AddReplicaSet(rs *v1beta1.ReplicaSet) {
	// check if any replicaset exist in this replicaset namespace yet
	if _, exists := rsc.namespaceRsUIDCache[rs.Namespace]; !exists {
		rsc.namespaceRsUIDCache[rs.Namespace] = make(map[types.UID]bool)
	}
	rsc.cachedReplicaSets[rs.UID] = newCachedReplicaSet(rs)
	rsc.namespaceRsUIDCache[rs.Namespace][rs.UID] = true
}

// Delete removes a replicaset from the cache
func (rsc *ReplicaSetCache) Delete(rs *v1beta1.ReplicaSet) {
	rsc.DeleteByKey(rs.UID)
}

// DeleteByKey removes a replicaset from the cache given a UID
func (rsc *ReplicaSetCache) DeleteByKey(rsUID types.UID) {
	namespace := rsc.cachedReplicaSets[rsUID].Namespace
	delete(rsc.namespaceRsUIDCache[namespace], rsUID)
	delete(rsc.cachedReplicaSets, rsUID)
}

// GetMatchingDeployment finds a matching replicaset given
// a namespace and a replicaSet name
func (rsc *ReplicaSetCache) GetMatchingDeployment(namespace string, replicaSetName string) (*string, types.UID) {
	var dpName *string
	var dpUID types.UID
	for rsUID := range rsc.namespaceRsUIDCache[namespace] {
		cachedRs := rsc.cachedReplicaSets[rsUID]
		if cachedRs.Name == replicaSetName {
			dpName = cachedRs.Deployment
			dpUID = cachedRs.DeploymentUID
			break
		}
	}
	return dpName, dpUID
}
