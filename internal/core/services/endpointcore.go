package services

import (
	"regexp"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// PortType represents the transport protocol used to communicate with this port
type PortType string

const (
	UDP     PortType = "UDP"
	TCP     PortType = "TCP"
	UNKNOWN PortType = "UNKNOWN"
)

//nolint:gochecknoglobals
var ipAddrRegexp = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)

var _ config.CustomConfigurable = &EndpointCore{}

// EndpointCore represents an exposed network port
type EndpointCore struct {
	ID ID `yaml:"id"`
	// A observer assigned name of the endpoint
	Name string `yaml:"name"`
	// The hostname/IP address of the endpoint
	Host string `yaml:"host"`
	// TCP or UDP
	PortType PortType `yaml:"port_type"`
	// The TCP/UDP port number of the endpoint
	Port uint16 `yaml:"port"`
	// The observer that discovered this endpoint
	DiscoveredBy  string                 `yaml:"discovered_by"`
	Configuration map[string]interface{} `yaml:"-"`
	// The type of monitor that this endpoint has requested.  This is populated
	// by observers that pull configuration directly from the platform they are
	// observing.
	MonitorType     string                 `yaml:"-"`
	extraDimensions map[string]string      `yaml:"-"`
	extraFields     map[string]interface{} `yaml:"-"`
}

// NewEndpointCore returns a new initialized endpoint core struct
func NewEndpointCore(id string, name string, discoveredBy string, dims map[string]string) *EndpointCore {
	if id == "" {
		// Observers must provide an ID or else they are majorly broken
		panic("EndpointCore cannot be created without an id")
	}

	ec := &EndpointCore{
		ID:              ID(id),
		Name:            name,
		DiscoveredBy:    discoveredBy,
		extraDimensions: dims,
		extraFields:     map[string]interface{}{},
	}

	return ec
}

// Core returns the EndpointCore since it will be embedded in an Endpoint
// instance
func (e *EndpointCore) Core() *EndpointCore {
	return e
}

// ENDPOINT_VAR(network_port): An alias for `port`
// ENDPOINT_VAR(ip_address): The IP address of the endpoint if the `host` is in
// the from of an IPv4 address

// DerivedFields returns aliased and computed variable fields for this endpoint
func (e *EndpointCore) DerivedFields() map[string]interface{} {
	out := map[string]interface{}{
		"network_port": e.Port,
	}
	if ipAddrRegexp.MatchString(e.Host) {
		out["ip_address"] = e.Host
	}
	return utils.MergeInterfaceMaps(utils.CloneInterfaceMap(e.extraFields), utils.StringMapToInterfaceMap(e.Dimensions()), out)
}

// ExtraConfig returns a map of values to be considered when configuring a monitor
func (e *EndpointCore) ExtraConfig() (map[string]interface{}, error) {
	return utils.MergeInterfaceMaps(
		map[string]interface{}{
			"host": e.Host,
			"port": e.Port,
			"name": utils.FirstNonEmpty(e.Name, string(e.ID)),
		}, e.Configuration), nil
}

// IsSelfConfigured tells whether this endpoint comes with enough configuration
// to run without being configured further.  This ultimately just means whether
// it specifies what type of monitor to use to monitor it.
func (e *EndpointCore) IsSelfConfigured() bool {
	return e.MonitorType != ""
}

// Dimensions returns a map of dimensions set on this endpoint
func (e *EndpointCore) Dimensions() map[string]string {
	return utils.CloneStringMap(e.extraDimensions)
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

func (e *EndpointCore) AddExtraField(name string, val interface{}) {
	e.extraFields[name] = val
}
