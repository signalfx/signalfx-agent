package utils

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type podsSet map[types.UID]bool

// PodCache is used for holding values we care about from a pod
// for quicker lookup than querying the API for them each time.
type PodCache struct {
	namespacePodUIDCache map[string]podsSet
	cachedPods           map[types.UID]*CachedPod
}

// CachedPod is used for holding only the necessary
type CachedPod struct {
	UID             types.UID
	LabelSet        labels.Set
	OwnerReferences []metav1.OwnerReference
	Namespace       string
	Tolerations     []v1.Toleration
}

func newCachedPod(pod *v1.Pod) *CachedPod {
	return &CachedPod{
		UID:             pod.UID,
		LabelSet:        labels.Set(pod.Labels),
		OwnerReferences: pod.OwnerReferences,
		Namespace:       pod.Namespace,
		Tolerations:     pod.Spec.Tolerations,
	}
}

// NewPodCache creates a new minimal pod cache
func NewPodCache() *PodCache {
	return &PodCache{
		namespacePodUIDCache: make(map[string]podsSet),
		cachedPods:           make(map[types.UID]*CachedPod),
	}
}

// IsCached checks if a pod was already in the cache, or if
// the mapped values have changed. Returns true if no change
func (pc *PodCache) IsCached(pod *v1.Pod) bool {
	cachedPod, exists := pc.cachedPods[pod.UID]
	labelSet := labels.Set(pod.Labels)
	return exists && reflect.DeepEqual(cachedPod.LabelSet, labelSet) &&
		(cachedPod.Namespace == pod.Namespace) &&
		(reflect.DeepEqual(cachedPod.OwnerReferences, pod.OwnerReferences))
}

// AddPod adds or updates a pod in cache
// This function should only be called after pc.IsCached
// to prevent unnecessary updates to the internal cache.
func (pc *PodCache) AddPod(pod *v1.Pod) {
	// check if any pods exist in this pods namespace
	if _, exists := pc.namespacePodUIDCache[pod.Namespace]; !exists {
		pc.namespacePodUIDCache[pod.Namespace] = make(map[types.UID]bool)
	}
	pc.namespacePodUIDCache[pod.Namespace][pod.UID] = true
	pc.cachedPods[pod.UID] = newCachedPod(pod)
}

// DeleteByKey removes a pod from the cache given a UID
func (pc *PodCache) DeleteByKey(key types.UID) {
	namespace := pc.cachedPods[key].Namespace
	delete(pc.namespacePodUIDCache[namespace], key)
	if len(pc.namespacePodUIDCache[namespace]) == 0 {
		delete(pc.namespacePodUIDCache, namespace)
	}
	delete(pc.cachedPods, key)
}

// GetLabels retrieves a pod's cached label set
func (pc *PodCache) GetLabels(key types.UID) labels.Set {
	return pc.cachedPods[key].LabelSet
}

// GetOwnerReferences retrieves a pod's cached owner references
func (pc *PodCache) GetOwnerReferences(key types.UID) []metav1.OwnerReference {
	return pc.cachedPods[key].OwnerReferences
}

// GetPodsInNamespace returns a list of pod UIDs given a namespace
func (pc *PodCache) GetPodsInNamespace(namespace string) []types.UID {
	var pods []types.UID
	if podsSet, exists := pc.namespacePodUIDCache[namespace]; exists {
		for podUID := range podsSet {
			pods = append(pods, podUID)
		}
	}
	return pods
}

// GetCachedPod returns a CachedPod object from the cache if it exists
func (pc *PodCache) GetCachedPod(podUID types.UID) *CachedPod {
	var cachedPod *CachedPod
	if pod, exists := pc.cachedPods[podUID]; exists {
		cachedPod = pod
	}
	return cachedPod
}
