package services

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/signalfx/neo-agent/core/config/types"
)

// PortType represents the transport protocol used to communicate with this port
type PortType string

const (
	// UDP port type
	UDP PortType = "UDP"
	// TCP port type
	TCP PortType = "TCP"
	// PRIVATE Port preference
)

var IPAddrRegexp = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

// EndpointCore represents an exposed network port
type EndpointCore struct {
	MID         ID        `yaml:"id"`
	Name        string    `yaml:"name"`
	Host        string    `yaml:"host"`
	PortType    PortType  `yaml:"portType"`
	Port        uint16    `yaml:"port"`
	MDiscovered time.Time `yaml:"discovered"`
	// The observer that discovered this endpoint
	MDiscoveredBy    string `yaml:"discoveredBy"`
	matchingMonitors map[types.MonitorID]bool
	MDimensions      map[string]string `yaml:"dimensions"`
	mlock            *sync.Mutex
}

// DerivedFields returns aliased and computed fields for this endpoint
func (e *EndpointCore) DerivedFields() map[string]interface{} {
	out := map[string]interface{}{
		"networkPort": e.Port,
	}
	if IPAddrRegexp.MatchString(e.Host) {
		out["ipAddress"] = e.Host
	}
	return out
}

// NewEndpointCore returns a new initialized endpoint core struct
func NewEndpointCore(id string, name string, discovered time.Time, discoveredBy string) *EndpointCore {
	return &EndpointCore{
		MID:              ID(id),
		Name:             name,
		MDiscovered:      discovered,
		MDiscoveredBy:    discoveredBy,
		matchingMonitors: make(map[types.MonitorID]bool),
		MDimensions:      make(map[string]string),
	}
}

// ID returns a unique id for this endpoint
func (e *EndpointCore) ID() ID {
	if len(e.MID) == 0 {
		e.MID = ID(fmt.Sprintf("%p", e))
	}
	return e.MID
}

// Hostname returns the host/ip address associated with this endpoint
func (e *EndpointCore) Hostname() string {
	return e.Host
}

// Discovered returns the time that this endpoint was first observed
func (e *EndpointCore) Discovered() time.Time {
	return e.MDiscovered
}

// DiscoveredBy returns the name of the observer that discovered this endpoint
func (e *EndpointCore) DiscoveredBy() string {
	return e.MDiscoveredBy
}

// Dimensions returns a map of dimensions set on this endpoint
func (e *EndpointCore) Dimensions() map[string]string {
	return e.MDimensions
}

// AddDimension adds a dimension to this endpoint
func (e *EndpointCore) AddDimension(k string, v string) {
	if e.MDimensions == nil {
		e.MDimensions = make(map[string]string)
	}

	e.MDimensions[k] = v
}

// RemoveDimension removes a dimension from this endpoint
func (e *EndpointCore) RemoveDimension(k string) {
	delete(e.MDimensions, k)
}

func (e *EndpointCore) String() string {
	return fmt.Sprintf("%#v", EndpointAsMap(e))
}

// MatchingMonitors returns a set of monitor ids that are monitoring this
// endpoint
func (e *EndpointCore) MatchingMonitors() map[types.MonitorID]bool {
	return e.matchingMonitors
}

func (e *EndpointCore) lock() {
	if e.mlock == nil {
		e.mlock = &sync.Mutex{}
	}
	e.mlock.Lock()
}

func (e *EndpointCore) unlock() {
	e.mlock.Unlock()
}

// AddMatchingMonitor adds a monitor id to the set of matched monitors
func (e *EndpointCore) AddMatchingMonitor(id types.MonitorID) {
	e.lock()
	defer e.unlock()

	if e.matchingMonitors == nil {
		e.matchingMonitors = make(map[types.MonitorID]bool)
	}

	e.matchingMonitors[id] = true
}

// RemoveMatchingMonitor removes a monitor id from the set of matched monitors
func (e *EndpointCore) RemoveMatchingMonitor(id types.MonitorID) {
	e.lock()
	defer e.unlock()

	delete(e.matchingMonitors, id)
}
