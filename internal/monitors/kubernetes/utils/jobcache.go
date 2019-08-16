package utils

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/types"
)

type jobSet map[types.UID]bool

// JobCache is used for holding values we care about from a job
// for quicker lookup than querying the API for them each time.
type JobCache struct {
	namespaceJobUIDCache map[string]jobSet
	cachedJobs           map[types.UID]*CachedJob
}

// NewJobCache creates a new minimal job cache
func NewJobCache() *JobCache {
	return &JobCache{
		namespaceJobUIDCache: make(map[string]jobSet),
		cachedJobs:           make(map[types.UID]*CachedJob),
	}
}

// CachedJob is used for holding only the neccesarry fields we need
// for label syncing
type CachedJob struct {
	UID        types.UID
	Name       string
	Namespace  string
	CronJob    *string
	CronJobUID types.UID
}

// newCachedJob checks if a job was created by a cronjob caches
// it if so. We only care about these types of jobs for linking
// pods to cronjobs directly.
func newCachedJob(job *batchv1.Job) *CachedJob {
	var cronJob *string
	var cronJobUID types.UID
	for _, or := range job.OwnerReferences {
		if or.Kind == "CronJob" {
			cronJob = &or.Name
			cronJobUID = or.UID
			break
		}
	}
	if cronJob != nil {
		return &CachedJob{
			UID:        job.UID,
			Name:       job.Name,
			Namespace:  job.Namespace,
			CronJob:    cronJob,
			CronJobUID: cronJobUID,
		}
	}

	return nil
}

// IsCached checks if a job is already in the cache or if any of the
// the cached fields have changed.
func (jc *JobCache) IsCached(job *batchv1.Job) bool {
	cachedJob, exists := jc.cachedJobs[job.UID]
	return exists &&
		(cachedJob.Name == job.Name) &&
		(cachedJob.Namespace == job.Namespace)
}

// AddJob adds or updates a job in cache
// This function should only be called after jc.IsCached
// to prevent unnecessary updates to the internal cache.
// Returns true if the job is added, false if it was ignored
func (jc *JobCache) AddJob(job *batchv1.Job) bool {
	// check if any jobs exist in this job namespace yet
	if _, exists := jc.namespaceJobUIDCache[job.Namespace]; !exists {
		jc.namespaceJobUIDCache[job.Namespace] = make(map[types.UID]bool)
	}
	cachedJob := newCachedJob(job)
	if cachedJob != nil {
		jc.cachedJobs[job.UID] = cachedJob
		jc.namespaceJobUIDCache[job.Namespace][job.UID] = true
		return true
	}
	return false
}

// DeleteByKey removes a job from the cache given a UID
func (jc *JobCache) DeleteByKey(jobUID types.UID) {
	namespace := jc.cachedJobs[jobUID].Namespace
	delete(jc.namespaceJobUIDCache[namespace], jobUID)
	if len(jc.namespaceJobUIDCache[namespace]) == 0 {
		delete(jc.namespaceJobUIDCache, namespace)
	}
	delete(jc.cachedJobs, jobUID)
}

// GetMatchingCronJob finds a matching cronjob given a namespace and job name
func (jc *JobCache) GetMatchingCronJob(namespace string, jobName string) (*string, types.UID) {
	var cjName *string
	var cjUID types.UID
	for jobUID := range jc.namespaceJobUIDCache[namespace] {
		cachedJob := jc.cachedJobs[jobUID]
		if cachedJob.Name == jobName {
			cjName = cachedJob.CronJob
			cjUID = cachedJob.CronJobUID
			break
		}
	}
	return cjName, cjUID

}
