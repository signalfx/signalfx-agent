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
	dpCacheLock sync.Mutex

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

	// tenantEnvironmentCache is the cache for environments related to containers/pods
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

// dpForService actually makes the datapoint that is put into the dp cache
func dpForService(service string) *datapoint.Datapoint {
	return sfxclient.Gauge(spanCorrelationMetricName, map[string]string{
		"sf_hasService": service,
		// Host dimensions get added in the writer just like datapoints
	}, 0)
}

// addServiceToDPCache creates a datapoint for the given service in the dpCache.
func (a *ActiveServiceTracker) addServiceToDPCache(service string) {
	a.dpCacheLock.Lock()
	defer a.dpCacheLock.Unlock()

	dp := dpForService(service)
	a.dpCache[service] = dp

	if a.newServiceCallback != nil {
		a.newServiceCallback(dp)
	}
}

// removeServiceFromDPCache removes the datapoint for the given service from the dpCache
func (a *ActiveServiceTracker) removeServiceFromDPCache(service string) {
	a.dpCacheLock.Lock()
	delete(a.dpCache, service)
	a.dpCacheLock.Unlock()
}

// CorrelationDatapoints returns a list of host correlation datapoints based on
// the spans sent through ProcessSpans
func (a *ActiveServiceTracker) CorrelationDatapoints() []*datapoint.Datapoint {
	a.dpCacheLock.Lock()
	defer a.dpCacheLock.Unlock()

	out := make([]*datapoint.Datapoint, 0, len(a.dpCache))
	for _, dp := range a.dpCache {
		out = append(out, dp)
	}
	return out
}

// LoadHostIDDimCorrelations asynchronously retrieves all known correlations from the backend
// for all known hostIDDims.  This allows the agent to timeout and manage correlation
// deletions on restart.
func (a *ActiveServiceTracker) LoadHostIDDimCorrelations() {
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
					// Note that only the value is set for the host service cache because we only track services for the host
					// therefore there we don't need to include the dim key and value on the cache key
					if isNew := a.hostServiceCache.UpdateOrCreate(&cacheKey{value: service}, a.timeNow()); isNew {
						if a.sendTraceHostCorrelationMetrics {
							// create datapoint for service
							a.addServiceToDPCache(service)
						}

						log.WithFields(log.Fields{
							"service": service,
						}).Debug("Tracking service name from trace span")
					}
				}
			}
			if environments, ok := correlations["sf_environments"]; ok {
				// Note that only the value is set for the host environment cache because we only track environments for the host
				// therefore there we don't need to include the dim key and value on the cache key
				for _, environment := range environments {
					if isNew := a.hostEnvironmentCache.UpdateOrCreate(&cacheKey{value: environment}, a.timeNow()); isNew {
						log.WithFields(log.Fields{
							"environment": environment,
						}).Debug("Tracking environment name from trace span")
					}
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
	a.LoadHostIDDimCorrelations()

	return a
}

// AddSpans accepts a list of trace spans and uses them to update the
// current list of active services.  This is thread-safe.
func (a *ActiveServiceTracker) AddSpans(ctx context.Context, spans []*trace.Span) {
	// Take current time once since this is a system call.
	now := a.timeNow()

	for i := range spans {
		a.processEnvironment(spans[i], now)
		a.processService(spans[i], now)
	}

	// Protected by lock above
	atomic.AddInt64(&a.spansProcessed, int64(len(spans)))
}

func (a *ActiveServiceTracker) processEnvironment(span *trace.Span, now time.Time) {
	if span.Tags == nil {
		return
	}
	environment, environmentFound := span.Tags["environment"]
	if !environmentFound || environment == "" {
		return
	}

	// update the environment for the hostIDDims
	// Note that only the value is set for the host environment cache because we only track environments for the host
	// therefore there we don't need to include the dim key and value on the cache key
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
			// Note that the value is not set on the cache key.  We only send the first environment received for a
			// given pod/container, and we never delete the values set on the container/pod dimension.
			// So we only need to cache the dim name and dim value that have been associated with an environment.
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
	// Note that only the value is set for the host service cache because we only track services for the host
	// therefore there we don't need to include the dim key and value on the cache key
	if isNew := a.hostServiceCache.UpdateOrCreate(&cacheKey{value: service}, now); isNew {
		// all of the host id dims need to be correlated with the service
		for dimName, dimValue := range a.hostIDDims {
			a.correlationClient.Correlate(&correlations.Correlation{
				Type:     correlations.Service,
				DimName:  dimName,
				DimValue: dimValue,
				Value:    service,
			})
		}

		if a.sendTraceHostCorrelationMetrics {
			// create datapoint for service
			a.addServiceToDPCache(service)
		}

		log.WithFields(log.Fields{
			"service": service,
		}).Debug("Tracking service name from trace span")
	}

	// container / pod level stuff (this should not directly affect the active service count)
	// this cache is necessary to identify services associated with a kubernetes pod or container id
	for _, dimName := range dimsToSyncSource {
		if dimValue, ok := span.Tags[dimName]; ok {
			// Note that the value is not set on the cache key.  We only send the first service received for a
			// given pod/container, and we never delete the values set on the container/pod dimension.
			// So we only need to cache the dim name and dim value that have been associated with a service.
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

// Purges caches on the ActiveServiceTracker
func (a *ActiveServiceTracker) Purge() {
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
		if a.sendTraceHostCorrelationMetrics {
			a.removeServiceFromDPCache(purged.value)
		}

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
			"environmentName": purged.value,
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
		sfxclient.Gauge("sfxagent.tracing_active_services", nil, a.hostServiceCache.GetActiveCount()),
		sfxclient.Cumulative("sfxagent.tracing_purged_services", nil, a.hostServiceCache.GetPurgedCount()),
		sfxclient.Gauge("sfxagent.tracing_active_environments", nil, a.hostEnvironmentCache.GetActiveCount()),
		sfxclient.Cumulative("sfxagent.tracing_purged_environments", nil, a.hostEnvironmentCache.GetPurgedCount()),
		sfxclient.CumulativeP("sfxagent.tracing_spans_processed", nil, &a.spansProcessed),
	}
}
