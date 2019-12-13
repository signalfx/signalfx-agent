package tracetracker

import (
	"container/list"
	"context"
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
	// How long to keep sending metrics for a particular service name after it
	// is last seen
	timeout time.Duration
	// A linked list of serviceNames sorted by time last seen
	serviceNameByTime *list.List
	// Which service names are active currently.  The value is an entry in the
	// serviceNameByTime linked list so that it can be quickly accessed and
	// moved to the back of the list.
	serviceNamesActive map[string]*list.Element
	// Datapoints don't change over time for a given service, so make them once
	// and stick them in here.
	dpCache map[string]*datapoint.Datapoint
	// A callback that gets called with the correlation datapoint when a
	// service is first seen
	newServiceCallback func(dp *datapoint.Datapoint)
	timeNow            func() time.Time

	// Internal metrics
	activeServiceCount int64
	purgedServiceCount int64
	spansProcessed     int64
}

type serviceStatus struct {
	LastSeen    time.Time
	ServiceName string
}

// New creates a new initialized service tracker
func New(timeout time.Duration, newServiceCallback func(dp *datapoint.Datapoint)) *ActiveServiceTracker {
	return &ActiveServiceTracker{
		timeout:            timeout,
		serviceNameByTime:  list.New(),
		serviceNamesActive: make(map[string]*list.Element),
		dpCache:            make(map[string]*datapoint.Datapoint),
		newServiceCallback: newServiceCallback,
		timeNow:            time.Now,
	}
}

// CorrelationDatapoints returns a list of host correlation datapoints based on
// the spans sent through ProcessSpans
func (a *ActiveServiceTracker) CorrelationDatapoints() []*datapoint.Datapoint {
	a.Lock()
	defer a.Unlock()

	// Get rid of everything that is old
	a.purgeOldServices()

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
	// Can't do anything if the spans don't have a local service name
	if span.LocalEndpoint == nil || span.LocalEndpoint.ServiceName == nil {
		return
	}

	service := *span.LocalEndpoint.ServiceName
	a.ensureServiceActive(service, now)
}

func (a *ActiveServiceTracker) ensureServiceActive(service string, now time.Time) {
	if timeElm, ok := a.serviceNamesActive[service]; ok {
		timeElm.Value.(*serviceStatus).LastSeen = now
		a.serviceNameByTime.MoveToFront(timeElm)
	} else {
		elm := a.serviceNameByTime.PushFront(&serviceStatus{
			LastSeen:    now,
			ServiceName: service,
		})
		a.serviceNamesActive[service] = elm
		dp := dpForService(service)
		a.dpCache[service] = dp

		atomic.AddInt64(&a.activeServiceCount, 1)

		if a.newServiceCallback != nil {
			a.newServiceCallback(dp)
		}
		log.WithFields(log.Fields{
			"service": service,
		}).Debug("Tracking service name from trace span")
	}
}

// Walks the serviceNameByTime list backwards and deletes until it finds an
// element that is not timed out.
func (a *ActiveServiceTracker) purgeOldServices() {
	now := a.timeNow()
	for {
		elm := a.serviceNameByTime.Back()
		if elm == nil {
			break
		}
		status := elm.Value.(*serviceStatus)
		// If this one isn't timed out, nothing else in the list is either.
		if now.Sub(status.LastSeen) < a.timeout {
			break
		}

		a.serviceNameByTime.Remove(elm)
		delete(a.serviceNamesActive, status.ServiceName)
		delete(a.dpCache, status.ServiceName)

		atomic.AddInt64(&a.activeServiceCount, -1)
		atomic.AddInt64(&a.purgedServiceCount, 1)

		log.WithFields(log.Fields{
			"serviceName": status.ServiceName,
		}).Debug("No longer tracking service name from trace span")
	}
}

// InternalMetrics returns datapoint describing the status of the tracker
func (a *ActiveServiceTracker) InternalMetrics() []*datapoint.Datapoint {
	return []*datapoint.Datapoint{
		sfxclient.Gauge("sfxagent.tracing_active_services", nil, atomic.LoadInt64(&a.activeServiceCount)),
		sfxclient.CumulativeP("sfxagent.tracing_purged_services", nil, &a.purgedServiceCount),
		sfxclient.CumulativeP("sfxagent.tracing_spans_processed", nil, &a.spansProcessed),
	}
}

func dpForService(service string) *datapoint.Datapoint {
	return sfxclient.Gauge(spanCorrelationMetricName, map[string]string{
		"sf_hasService": service,
		// Host dimensions get added in the writer just like datapoints
	}, 0)
}
