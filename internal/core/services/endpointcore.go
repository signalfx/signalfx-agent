package services

import (
	"regexp"

	"github.com/signalfx/signalfx-agent/internal/utils"
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

var ipAddrRegexp = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

// EndpointCore represents an exposed network port
type EndpointCore struct {
	ID       ID       `yaml:"id"`
	Name     string   `yaml:"name"`
	Host     string   `yaml:"host"`
	PortType PortType `yaml:"portType"`
	Port     uint16   `yaml:"port"`
	// The observer that discovered this endpoint
	DiscoveredBy  string                 `yaml:"discoveredBy"`
	Configuration map[string]interface{} `yaml:"configuration"`
	// The type of monitor that this endpoint has requested.  This is populated
	// by observers that pull configuration directly from the platform they are
	// observing.
	MonitorType     string            `yaml:"monitorType"`
	extraDimensions map[string]string `yaml:"dimensions"`
}

// Core returns the EndpointCore since it will be embedded in an Endpoint
// instance
func (e *EndpointCore) Core() *EndpointCore {
	return e
}

// DerivedFields returns aliased and computed fields for this endpoint
func (e *EndpointCore) DerivedFields() map[string]interface{} {
	out := map[string]interface{}{
		"networkPort": e.Port,
	}
	if ipAddrRegexp.MatchString(e.Host) {
		out["ipAddress"] = e.Host
	}
	return out
}

// NewEndpointCore returns a new initialized endpoint core struct
func NewEndpointCore(id string, name string, discoveredBy string) *EndpointCore {
	if id == "" {
		// Observers must provide an ID or else they are majorly broken
		panic("EndpointCore cannot be created without an id")
	}

	ec := &EndpointCore{
		ID:           ID(id),
		Name:         name,
		DiscoveredBy: discoveredBy,
	}

	return ec
}

// ExtraConfig returns a map of values to be considered when configuring a monitor
func (e *EndpointCore) ExtraConfig() map[string]interface{} {
	return utils.MergeInterfaceMaps(
		map[string]interface{}{
			"host": e.Host,
			"port": e.Port,
			"name": utils.FirstNonEmpty(e.Name, string(e.ID)),
		}, e.Configuration)
}

// IsSelfConfigured tells whether this endpoint comes with enough configuration
// to run without being configured further.  This ultimately just means whether
// it specifies what type of monitor to use to monitor it.
func (e *EndpointCore) IsSelfConfigured() bool {
	return e.MonitorType != ""
}

// Dimensions returns a map of dimensions set on this endpoint
func (e *EndpointCore) Dimensions() map[string]string {
	return e.extraDimensions
}

// AddDimension adds a dimension to this endpoint
func (e *EndpointCore) AddDimension(k string, v string) {
	if e.extraDimensions == nil {
		e.extraDimensions = make(map[string]string)
	}

	e.extraDimensions[k] = v
}

// RemoveDimension removes a dimension from this endpoint
func (e *EndpointCore) RemoveDimension(k string) {
	delete(e.extraDimensions, k)
}
