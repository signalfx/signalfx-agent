package tracetracker

import (
	"context"
	"github.com/signalfx/signalfx-agent/pkg/core/writer/correlations"
	"sync"
	"sync/atomic"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"
	"github.com/signalfx/golib/v3/trace"
	log "github.com/sirupsen/logrus"
)

const spanCorrelationMetricName = "sf.int.service.heartbeat"

// ActiveServiceTracker keeps track of which services are seen in the trace
// spans passed through ProcessSpans.  It supports expiry of service names if
// they are not seen for a certain amount of time.
type ActiveServiceTracker struct {
	sync.Mutex

	// environment is the default environment value to associate with the host
	environment string

	// hostIDDims is the map of key/values discovered by the agent that identify the host
	hostIDDims map[string]string

	// keyCache of services and environment associations beyond the host level
	// NOTE: if we ever care about specific stats, we could always move these to separate caches later
	keyCache *TimeoutCache

	// hostServiceCache is a cache of services associated with the host
	hostServiceCache *TimeoutCache

	// hostEnvironmentCache is a cache of environments associated with the host
	hostEnvironmentCache *TimeoutCache

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

// New creates a new initialized service tracker
func New(timeout time.Duration, correlationClient correlations.CorrelationClient, hostIDDims map[string]string, environment string, newServiceCallback func(dp *datapoint.Datapoint)) *ActiveServiceTracker {
	return &ActiveServiceTracker{
		environment:          environment,
		hostIDDims:           hostIDDims,
		hostServiceCache:     NewTimeoutCache(timeout),
		hostEnvironmentCache: NewTimeoutCache(timeout),
		keyCache:             NewTimeoutCache(timeout),
		dpCache:              make(map[string]*datapoint.Datapoint),
		newServiceCallback:   newServiceCallback,
		correlationClient:    correlationClient,
		timeNow:              time.Now,
	}
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
		a.processSpan(spans[i], now)
	}

	// Protected by lock above
	a.spansProcessed += int64(len(spans))
}

func (a *ActiveServiceTracker) processSpan(span *trace.Span, now time.Time) {
	environment, environmentFound := span.Tags["environment"]
	if !environmentFound {
		environment = a.environment
	}

	// update the environment for the hostIDDims
	if isNew := a.hostEnvironmentCache.UpdateOrCreate(&CacheKey{environment: environment}, now); isNew {
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.AcceptCorrelation(&correlations.Correlation{
				Type:      correlations.Environment,
				Operation: correlations.Put,
				DimName:   dimName,
				DimValue:  dimValue,
				Value:     environment,
			})
		}
	}

	// container / pod level stuff
	// this cache is necessary to identify environments associated with a kubernetes pod or container id
	for _, dimName := range dimsToSyncSource {
		if dimValue, ok := span.Tags[dimName]; ok {
			if isNew := a.keyCache.UpdateOrCreate(&CacheKey{dimName: dimName, dimValue: dimValue, environment: environment}, now); isNew {
				a.correlationClient.AcceptCorrelation(&correlations.Correlation{
					Type:      correlations.Environment,
					Operation: correlations.Put,
					DimName:   dimName,
					DimValue:  dimValue,
					Value:     environment,
				})
			}

			if environment != a.environment {
				// if a span comes through with an environment, we add it to the cache.
				// we must also add the default environment so that a later span without
				// an environment tag doesn't overwrite the environment property with the
				// default environment for the same k8s/container id
				a.keyCache.UpdateOrCreate(&CacheKey{dimName: dimName, dimValue: dimValue, environment: a.environment}, now)
			}
		}
	}

	// Can't do anything if the spans don't have a local service name
	if span.LocalEndpoint == nil || span.LocalEndpoint.ServiceName == nil {
		return
	}
	service := *span.LocalEndpoint.ServiceName

	// Handle host level service and environment correlation
	if isNew := a.ensureServiceActive(&CacheKey{service: service}, now); isNew {
		// all of the host id dims need to be correlated with the service
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.AcceptCorrelation(&correlations.Correlation{
				Type:      correlations.Service,
				Operation: correlations.Put,
				DimName:   dimName,
				DimValue:  dimValue,
				Value:     service,
			})
		}
	}

	// container / pod level stuff (this should not directly affect the active service count)
	// this cache is necessary to identify services associated with a kubernetes pod or container id
	for _, dimName := range dimsToSyncSource {
		if dimValue, ok := span.Tags[dimName]; ok {
			if isNew := a.keyCache.UpdateOrCreate(&CacheKey{dimName: dimName, dimValue: dimValue, service: service}, now); isNew {
				a.correlationClient.AcceptCorrelation(&correlations.Correlation{
					Type:      correlations.Service,
					Operation: correlations.Put,
					DimName:   dimName,
					DimValue:  dimValue,
					Value:     service,
				})
			}
		}
	}
}

func (a *ActiveServiceTracker) ensureServiceActive(key *CacheKey, now time.Time) bool {
	isNew := a.hostServiceCache.UpdateOrCreate(key, now)
	if isNew {
		dp := dpForService(key.service)
		a.dpCache[key.service] = dp

		if a.newServiceCallback != nil {
			a.newServiceCallback(dp)
		}
		log.WithFields(log.Fields{
			"service": key.service,
		}).Debug("Tracking service name from trace span")
	}
	return isNew
}

// Purges caches on the ActiveServiceTracker
func (a *ActiveServiceTracker) Purge() {
	a.Lock()
	defer a.Unlock()
	now := a.timeNow()
	a.hostServiceCache.PurgeOld(now, func(purged *CacheKey) {
		// delete the correlation from all host id dims
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.AcceptCorrelation(&correlations.Correlation{
				Type:      correlations.Service,
				Operation: correlations.Delete,
				DimName:   dimName,
				DimValue:  dimValue,
				Value:     purged.service,
			})
		}
		// remove host/service correlation metric from tracker
		delete(a.dpCache, purged.service)
		log.WithFields(log.Fields{
			"serviceName": purged.service,
		}).Debug("No longer tracking service name from trace span")
	})
	a.hostEnvironmentCache.PurgeOld(now, func(purged *CacheKey) {
		// delete the correlation from all host id dims
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.AcceptCorrelation(&correlations.Correlation{
				Type:      correlations.Environment,
				Operation: correlations.Delete,
				DimName:   dimName,
				DimValue:  dimValue,
				Value:     purged.service,
			})
		}
		log.WithFields(log.Fields{
			"envrionmentName": purged.environment,
		}).Debug("No longer tracking environment name from trace span")
	})

	// Purge the cache, but don't do the deletion for container id and pod id.
	// These values aren't expected to change, and can be overwritten.
	// The onPurge() function doesn't need to do anything
	a.keyCache.PurgeOld(now, func(purged *CacheKey) {})
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
