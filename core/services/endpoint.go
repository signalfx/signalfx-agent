package services

import (
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

//ID uniquely identifies a service instance
type ID string

// Endpoint is the generic interface that all types of service instances should
// implement.  All consumers of services should use this interface only.
type Endpoint interface {
	// ID should be unique across all endpoints, determined by the discovering observer
	ID() ID
	// EnsureID should set an ID on the endpoint if one doesn't already exist
	EnsureID()
	// Hostname is the hostname or IP address of the endpoint
	Hostname() string
	// Discovered is the time that the endpoint was discovered by the agent
	Discovered() time.Time
	// DiscoveredBy is the name of the observer that discovered this endpoint
	DiscoveredBy() string
	// String is just the string representation of the endpoint
	String() string

	// Dimensions that are specific to this endpoint (e.g. container name)
	Dimensions() map[string]string
	// AddDimension adds a single dimension to the endpoint
	AddDimension(string, string)
	// RemoveDimension removes a single dimension from the endpoint
	RemoveDimension(string)

	// AddMatchingMonitor will add metadata about what monitors the endpoint
	// has been matched to.  This is useful for generic monitor helpers that could
	// receive multiple types of endpoints (e.g. GenericJMX).
	AddMatchingMonitor(types.MonitorID)
	// RemoveMatchingMonitor reverses what AddMatchingMonitor does
	RemoveMatchingMonitor(types.MonitorID)
	// MatchingMonitors returns a set of the currently matched monitors
	MatchingMonitors() map[types.MonitorID]bool
}

// HasDerivedFields is an interface with a single method that can be called to
// get fields that are derived from a service.  This is useful for things like
// aliased fields or computed fields.
type HasDerivedFields interface {
	DerivedFields() map[string]interface{}
}

// EndpointAsMap converts an endpoint to a map that contains all of the
// information about the endpoint.  This makes it easy to use endpoints in
// evaluating rules as well as in collectd templates.
func EndpointAsMap(endpoint Endpoint) map[string]interface{} {
	asMap, err := utils.ConvertToMapViaYAML(endpoint)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"endpoint": spew.Sdump(endpoint),
		}).Error("Could not convert endpoint to map")
		return nil
	}

	if asMap == nil {
		return nil
	}

	if df, ok := endpoint.(HasDerivedFields); ok {
		return utils.MergeInterfaceMaps(asMap, df.DerivedFields())
	}
	return asMap
}

// EndpointsAsSliceOfMap takes a slice of endpoint types and returns a slice of
// the result of mapping each endpoint through EndpointAsMap.  Panics if
// endpoints isn't a slice.
func EndpointsAsSliceOfMap(endpoints interface{}) []map[string]interface{} {
	val := reflect.ValueOf(endpoints)
	out := make([]map[string]interface{}, val.Len(), val.Len())
	for i := 0; i < val.Len(); i++ {
		out[i] = EndpointAsMap(val.Index(i).Interface().(Endpoint))
	}
	return out
}

/*func MergeCommonConfigToEndpoint(common Endpoint, endpoint Endpoint) Endpoint {

}*/
