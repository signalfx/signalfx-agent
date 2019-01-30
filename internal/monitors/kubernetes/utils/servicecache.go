package utils

import (
	"reflect"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type servicesSet map[types.UID]bool

// ServiceCache is used for holding values we care about from a pod
// for quicker lookup than querying the API for them each time.
type ServiceCache struct {
	namespaceSvcUIDCache map[string]servicesSet
	cachedServices       map[types.UID]*CachedService
}

// NewServiceCache creates a new minimal pod cache
func NewServiceCache() *ServiceCache {
	return &ServiceCache{
		namespaceSvcUIDCache: make(map[string]servicesSet),
		cachedServices:       make(map[types.UID]*CachedService),
	}
}

// CachedService is used for holding only the neccesarry fields we need
// for label syncing
type CachedService struct {
	UID         types.UID
	Name        string
	Namespace   string
	Selector    labels.Selector
	matchedPods podsSet
}

func newCachedService(svc *v1.Service) *CachedService {
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()
	if selector.Empty() {
		return nil
	}

	return &CachedService{
		UID:         svc.UID,
		Name:        svc.Name,
		Namespace:   svc.Namespace,
		Selector:    labels.Set(svc.Spec.Selector).AsSelectorPreValidated(),
		matchedPods: make(map[types.UID]bool),
	}
}

// IsCached checks if a service is already in the cache or if any of the
// the cached fields have changed.
func (sc *ServiceCache) IsCached(svc *v1.Service) bool {
	cachedService, exists := sc.cachedServices[svc.UID]
	selector := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()

	return exists &&
		(reflect.DeepEqual(cachedService.Selector, selector)) &&
		(cachedService.Name == svc.Name) &&
		(cachedService.Namespace == svc.Namespace)
}

// AddService adds or updates a service in cache
// This function should only be called after sc.IsCached
// to prevent unneccesary updates to the internal cache.
// Returns true if the service is added, false if it was ignored
func (sc *ServiceCache) AddService(svc *v1.Service) bool {
	// check if any services exist in this services namespace yet
	if _, exists := sc.namespaceSvcUIDCache[svc.Namespace]; !exists {
		sc.namespaceSvcUIDCache[svc.Namespace] = make(map[types.UID]bool)
	}
	cachedService := newCachedService(svc)
	// empty selectors match no pods so no need to cache these services
	if cachedService != nil {
		sc.cachedServices[svc.UID] = newCachedService(svc)
		sc.namespaceSvcUIDCache[svc.Namespace][svc.UID] = true
		return true
	}
	return false
}

// Delete removes a service from the cache
func (sc *ServiceCache) Delete(svc *v1.Service) {
	sc.DeleteByKey(svc.UID)
}

// DeleteByKey removes a service from the cache given a UID
// Returns pods that were previously matched by this service
// so their properties may be updated accordingly
func (sc *ServiceCache) DeleteByKey(svcUID types.UID) []types.UID {
	var pods []types.UID
	for podUID := range sc.cachedServices[svcUID].matchedPods {
		pods = append(pods, podUID)
	}
	namespace := sc.cachedServices[svcUID].Namespace
	delete(sc.namespaceSvcUIDCache[namespace], svcUID)
	delete(sc.cachedServices, svcUID)

	return pods
}

// GetMatchingServices returns a list of service names that match the given
// pod, given the services are in the cache already
func (sc *ServiceCache) GetMatchingServices(cachedPod *CachedPod) []string {
	var services []string
	// only check services in same namespace as pod
	for svcUID := range sc.namespaceSvcUIDCache[cachedPod.Namespace] {
		if sc.cachedServices[svcUID].Selector.Matches(cachedPod.LabelSet) {
			sc.cachedServices[svcUID].matchedPods[cachedPod.UID] = true
			services = append(services, sc.cachedServices[svcUID].Name)
		}
	}
	return services
}
