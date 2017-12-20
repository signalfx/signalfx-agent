package config

import (
	"net/url"
	"reflect"

	"github.com/signalfx/neo-agent/core/config/stores"
	"github.com/signalfx/neo-agent/core/filters"
)

// MonitorConfig is used to configure monitor instances.  One instance of
// MonitorConfig may be used to configure multiple monitor instances.  If a
// monitor's discovery rule does not match any discovered services, the monitor
// will not run.
type MonitorConfig struct {
	Type string `yaml:"type,omitempty"`
	// DiscoveryRule is what is used to auto discover service endpoints
	DiscoveryRule string `yaml:"discoveryRule,omitempty"`
	// ExtraDimensions specific to this monitor
	ExtraDimensions map[string]string `yaml:"dimensions" default:"{}"`
	// IntervalSeconds will default to the top-level IntervalSeconds value if unset or 0
	IntervalSeconds int `yaml:"intervalSeconds,omitempty" default:"0"`
	// Solo, if set to true, make this monitor the only one configured.  This
	// is useful for testing config in isolation without having to delete a
	// bunch of other stuff from the config file.
	Solo bool `yaml:"solo,omitempty" default:"false"`
	// OtherConfig is everything else that is custom to a particular monitor
	OtherConfig map[string]interface{} `yaml:",inline" default:"{}" json:"-"`
	// ValidationError is where a message concerning validation issues can go
	// so that diagnostics can output it.
	ValidationError string `yaml:"-"`
	// The remaining are propagated from the top-level config and cannot be set
	// by the user directly on the monitor
	IngestURL                 *url.URL           `yaml:"-"`
	SignalFxAccessToken       string             `yaml:"-"`
	Hostname                  string             `yaml:"-"`
	Filter                    *filters.FilterSet `yaml:"-"`
	ProcFSPath                string             `yaml:"-"`
	MetaStore                 *stores.MetaStore  `yaml:"-"`
	CollectdConf              *CollectdConfig    `yaml:"-"`
	InternalMetricsSocketPath string             `yaml:"-"`
}

// Equals tests if two monitor configs are sufficiently equal to each other.
// Two monitors should only be equal if it doesn't make sense for two
// configurations to be active at the same time.
func (mc *MonitorConfig) Equals(other *MonitorConfig) bool {
	return mc.Type == other.Type && mc.DiscoveryRule == other.DiscoveryRule &&
		reflect.DeepEqual(mc.OtherConfig, other.OtherConfig)
}

// ExtraConfig returns generic config as a map
func (mc *MonitorConfig) ExtraConfig() map[string]interface{} {
	return mc.OtherConfig
}

// HasAutoDiscovery returns whether the monitor is static (i.e. doesn't rely on
// autodiscovered services and is manually configured) or dynamic.
func (mc *MonitorConfig) HasAutoDiscovery() bool {
	return mc.DiscoveryRule != ""
}

// CoreConfig provides a way of getting the MonitorConfig when embedded in a
// struct that is referenced through a more generic interface.
func (mc *MonitorConfig) CoreConfig() *MonitorConfig {
	return mc
}

// MonitorCustomConfig represents monitor-specific configuration that doesn't
// appear in the MonitorConfig struct.
type MonitorCustomConfig interface {
	CoreConfig() *MonitorConfig
}
