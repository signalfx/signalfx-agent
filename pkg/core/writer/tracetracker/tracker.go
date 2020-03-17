package tracetracker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/signalfx/golib/v3/trace"
	"github.com/signalfx/signalfx-agent/pkg/core/writer/correlations"
	log "github.com/sirupsen/logrus"
)

const spanCorrelationMetricName = "sf.int.service.heartbeat"

// ActiveServiceTracker keeps track of which services are seen in the trace
// spans passed through ProcessSpans.  It supports expiry of service names if
// they are not seen for a certain amount of time.
type ActiveServiceTracker struct {
	sync.Mutex

	// hostIDDims is the map of key/values discovered by the agent that identify the host
	hostIDDims map[string]string

	// sendTraceHostCorrelationMetrics turns metric emission on and off
	sendTraceHostCorrelationMetrics bool

	// hostServiceCache is a cache of services associated with the host
	hostServiceCache *TimeoutCache

	// hostEnvironmentCache is a cache of environments associated with the host
	hostEnvironmentCache *TimeoutCache

	// tenantServiceCache is the cache for services related to containers/pods
	tenantServiceCache *TimeoutCache

	// tenantEnvrionmentCache is the cache for environments related to containers/pods
	tenantEnvironmentCache *TimeoutCache

	// Datapoints don't change over time for a given service, so make them once
	// and stick them in here.
	dpCache map[string]*datapoint.Datapoint

	// A callback that gets called with the correlation datapoint when a
	// service is first seen
	newServiceCallback func(dp *datapoint.Datapoint)
	timeNow            func() time.Time

	// correlationClient is the client used for updating infrastructure correlation properties
	correlationClient correlations.CorrelationClient

	// Internal metrics
	spansProcessed int64
}

// LoadCorrelations asynchronously retrieves all known correlations from the backend
// for all known hostIDDims.  This allows the agent to timeout and manage correlation
// deletions on restart.
func (a *ActiveServiceTracker) LoadCorrelations() {
	// asynchronously fetch all services and environments for each hostIDDim at startup
	for dimName, dimValue := range a.hostIDDims {
		dimName := dimName
		dimValue := dimValue
		a.correlationClient.Get(dimName, dimValue, func(correlations map[string][]string, err error) {
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"dim":   dimName,
					"value": dimValue,
				}).Error("Unable to unmarshall correlations for dimension")
				return
			}
			if services, ok := correlations["sf_services"]; ok {
				for _, service := range services {
					a.Lock() // lock before accessing services cache and dpcache
					a.ensureServiceActive(&cacheKey{dimName: dimName, dimValue: dimValue, value: service}, a.timeNow())
					a.Unlock()
				}
			}
			if environments, ok := correlations["sf_environments"]; ok {
				for _, environment := range environments {
					a.Lock() // lock before accessing environments cache
					a.hostEnvironmentCache.UpdateOrCreate(&cacheKey{dimName: dimName, dimValue: dimValue, value: environment}, a.timeNow())
					a.Unlock()
				}
			}
		})
	}
}

// New creates a new initialized service tracker
func New(timeout time.Duration, correlationClient correlations.CorrelationClient, hostIDDims map[string]string, sendTraceHostCorrelationMetrics bool, newServiceCallback func(dp *datapoint.Datapoint)) *ActiveServiceTracker {
	a := &ActiveServiceTracker{
		hostIDDims:                      hostIDDims,
		hostServiceCache:                NewTimeoutCache(timeout),
		hostEnvironmentCache:            NewTimeoutCache(timeout),
		tenantServiceCache:              NewTimeoutCache(timeout),
		tenantEnvironmentCache:          NewTimeoutCache(timeout),
		dpCache:                         make(map[string]*datapoint.Datapoint),
		newServiceCallback:              newServiceCallback,
		correlationClient:               correlationClient,
		sendTraceHostCorrelationMetrics: sendTraceHostCorrelationMetrics,
		timeNow:                         time.Now,
	}
	a.LoadCorrelations()

	return a
}

// CorrelationDatapoints returns a list of host correlation datapoints based on
// the spans sent through ProcessSpans
func (a *ActiveServiceTracker) CorrelationDatapoints() []*datapoint.Datapoint {
	a.Lock()
	defer a.Unlock()

	out := make([]*datapoint.Datapoint, 0, len(a.dpCache))
	for _, dp := range a.dpCache {
		out = append(out, dp)
	}
	return out
}

// AddSpans accepts a list of trace spans and uses them to update the
// current list of active services.  This is thread-safe.
func (a *ActiveServiceTracker) AddSpans(ctx context.Context, spans []*trace.Span) {
	// Take current time once since this is a system call.
	now := a.timeNow()

	a.Lock()
	defer a.Unlock()

	for i := range spans {
		a.processEnvironment(spans[i], now)
		a.processService(spans[i], now)
	}

	// Protected by lock above
	a.spansProcessed += int64(len(spans))
}

func (a *ActiveServiceTracker) processEnvironment(span *trace.Span, now time.Time) {
	if span.Tags == nil {
		return
	}
	environment, envrionmentFound := span.Tags["environment"]
	if !envrionmentFound || environment == "" {
		return
	}

	// update the environment for the hostIDDims
	if isNew := a.hostEnvironmentCache.UpdateOrCreate(&cacheKey{value: environment}, now); isNew {
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.Correlate(&correlations.Correlation{
				Type:     correlations.Environment,
				DimName:  dimName,
				DimValue: dimValue,
				Value:    environment,
			})
		}
		log.WithFields(log.Fields{
			"environment": environment,
		}).Debug("Tracking environment name from trace span")
	}

	// container / pod level stuff
	// this cache is necessary to identify environments associated with a kubernetes pod or container id
	for _, dimName := range dimsToSyncSource {
		if dimValue, ok := span.Tags[dimName]; ok {
			if isNew := a.tenantEnvironmentCache.UpdateOrCreate(&cacheKey{dimName: dimName, dimValue: dimValue}, now); isNew {
				a.correlationClient.Correlate(&correlations.Correlation{
					Type:     correlations.Environment,
					DimName:  dimName,
					DimValue: dimValue,
					Value:    environment,
				})
			}
		}
	}
}

func (a *ActiveServiceTracker) processService(span *trace.Span, now time.Time) {
	// Can't do anything if the spans don't have a local service name
	if span.LocalEndpoint == nil || span.LocalEndpoint.ServiceName == nil || *span.LocalEndpoint.ServiceName == "" {
		return
	}
	service := *span.LocalEndpoint.ServiceName

	// Handle host level service and environment correlation
	if isNew := a.ensureServiceActive(&cacheKey{value: service}, now); isNew {
		// all of the host id dims need to be correlated with the service
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.Correlate(&correlations.Correlation{
				Type:     correlations.Service,
				DimName:  dimName,
				DimValue: dimValue,
				Value:    service,
			})
		}
	}

	// container / pod level stuff (this should not directly affect the active service count)
	// this cache is necessary to identify services associated with a kubernetes pod or container id
	for _, dimName := range dimsToSyncSource {
		if dimValue, ok := span.Tags[dimName]; ok {
			if isNew := a.tenantServiceCache.UpdateOrCreate(&cacheKey{dimName: dimName, dimValue: dimValue}, now); isNew {
				a.correlationClient.Correlate(&correlations.Correlation{
					Type:     correlations.Service,
					DimName:  dimName,
					DimValue: dimValue,
					Value:    service,
				})
			}
		}
	}
}

func (a *ActiveServiceTracker) ensureServiceActive(key *cacheKey, now time.Time) bool {
	isNew := a.hostServiceCache.UpdateOrCreate(key, now)
	if isNew {
		if a.sendTraceHostCorrelationMetrics {
			// only add to the dp cache if we're sending host/service correlation metrics
			dp := dpForService(key.value)
			a.dpCache[key.value] = dp

			if a.newServiceCallback != nil {
				a.newServiceCallback(dp)
			}
		}
		log.WithFields(log.Fields{
			"service": key.value,
		}).Debug("Tracking service name from trace span")
	}
	return isNew
}

// Purges caches on the ActiveServiceTracker
func (a *ActiveServiceTracker) Purge() {
	a.Lock()
	defer a.Unlock()
	now := a.timeNow()
	a.hostServiceCache.PurgeOld(now, func(purged *cacheKey) {
		// delete the correlation from all host id dims
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.Delete(&correlations.Correlation{
				Type:     correlations.Service,
				DimName:  dimName,
				DimValue: dimValue,
				Value:    purged.value,
			})
		}
		// remove host/service correlation metric from tracker
		delete(a.dpCache, purged.value)
		log.WithFields(log.Fields{
			"serviceName": purged.value,
		}).Debug("No longer tracking service name from trace span")
	})
	a.hostEnvironmentCache.PurgeOld(now, func(purged *cacheKey) {
		// delete the correlation from all host id dims
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.Delete(&correlations.Correlation{
				Type:     correlations.Environment,
				DimName:  dimName,
				DimValue: dimValue,
				Value:    purged.value,
			})
		}
		log.WithFields(log.Fields{
			"envrionmentName": purged.value,
		}).Debug("No longer tracking environment name from trace span")
	})

	// Purge the caches for containers and pods, but don't do the deletions.
	// These values aren't expected to change, and can be overwritten.
	// The onPurge() function doesn't need to do anything
	a.tenantServiceCache.PurgeOld(now, func(purged *cacheKey) {})
	a.tenantEnvironmentCache.PurgeOld(now, func(purged *cacheKey) {})
}

// InternalMetrics returns datapoint describing the status of the tracker
func (a *ActiveServiceTracker) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Gauge("sfxagent.tracing_active_services", nil, atomic.LoadInt64(&a.hostServiceCache.ActiveCount)),
		sfxclient.CumulativeP("sfxagent.tracing_purged_services", nil, &a.hostServiceCache.PurgedCount),
		sfxclient.Gauge("sfxagent.tracing_active_environments", nil, atomic.LoadInt64(&a.hostEnvironmentCache.ActiveCount)),
		sfxclient.CumulativeP("sfxagent.tracing_purged_environments", nil, &a.hostEnvironmentCache.PurgedCount),
		sfxclient.CumulativeP("sfxagent.tracing_spans_processed", nil, &a.spansProcessed),
	}
}

func dpForService(service string) *datapoint.Datapoint {
	return sfxclient.Gauge(spanCorrelationMetricName, map[string]string{
		"sf_hasService": service,
		// Host dimensions get added in the writer just like datapoints
	}, 0)
}
