package config

import (
	"fmt"
	"net/url"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config/stores"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/core/filters"
	"github.com/signalfx/neo-agent/core/services"
)

// MonitorConfig configures a given monitor instance.  There is a 1-1
// correspondance between monitor config and monitor instances.  There will
// never be more monitor instances than there are monitor configurations, since
// all services that match the same discovery rule will be added to the same
// monitor instance.  If a monitor's discovery rule does not match any
// discovered services, the monitor will not run.
type MonitorConfig struct {
	Type string `yaml:"type,omitempty"`
	// ID can be used to uniquely identify monitors so that they can be
	// reconfigured in place instead of destroyed and recreated
	ID types.MonitorID `yaml:"id,omitempty"`
	// DiscoveryRule is what is used to auto discover service endpoints
	DiscoveryRule string `yaml:"discoveryRule,omitempty"`
	// ExtraDimensions specific to this monitor
	ExtraDimensions map[string]string `yaml:"dimensions,omitempty" default:"{}"`
	// K8s pod label keys to send as dimensions
	K8sLabelDimensions []string `yaml:"labelDimensions,omitempty" default:"[]"`
	// IntervalSeconds will default to the top-level IntervalSeconds value if unset or 0
	IntervalSeconds int `yaml:"intervalSeconds,omitempty" default:"0"`
	// Solo, if set to true, make this monitor the only one configured.  This
	// is useful for testing config in isolation without having to delete a
	// bunch of other stuff.
	Solo bool `yaml:"solo,omitempty" default:"false"`
	// OtherConfig is everything else that is custom to a particular monitor
	OtherConfig map[string]interface{} `yaml:",inline" default:"[]" json:"-"`
	// ValidationError is where a message concerning validation issues can go
	// so that diagnostics can output it.
	ValidationError string `yaml:"-"`
	// The remaining are propagated from the top-level config and cannot be set
	// by the user directly on the monitor
	IngestURL           *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	Hostname            string             `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
	ProcFSPath          string             `yaml:"-"`
	MetaStore           *stores.MetaStore  `yaml:"-"`
	CollectdConf        *CollectdConfig    `yaml:"-"`
}

// GetOtherConfig returns generic config as a map
func (mc *MonitorConfig) GetOtherConfig() map[string]interface{} {
	return mc.OtherConfig
}

// HasAutoDiscovery returns whether the monitor is static (i.e. doesn't rely on
// autodiscovered services and is manually configured) or dynamic.
func (mc *MonitorConfig) HasAutoDiscovery() bool {
	return mc.DiscoveryRule != ""
}

func (mc *MonitorConfig) ensureID(generator func(string) int) {
	if len(mc.ID) == 0 {
		mc.ID = types.MonitorID(fmt.Sprintf("%s-%d", mc.Type, generator(mc.Type)))
	}
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

// ServiceEndpointsFromConfig returns the manually defined service endpoints in
// a monitor config.  Returns nil if ServiceEndpoints is invalidly defined on a
// monitor's custom config and returns an empty slice if it is either not
// defined at all or properly defined but empty.
func ServiceEndpointsFromConfig(conf MonitorCustomConfig) []services.Endpoint {
	val := reflect.Indirect(reflect.ValueOf(conf))

	seField := val.FieldByName("ServiceEndpoints")
	if !seField.IsValid() {
		return make([]services.Endpoint, 0)
	}

	if seField.Kind() != reflect.Slice {
		log.WithFields(log.Fields{
			"type": seField.Type(),
		}).Error("ServiceEndpoints on monitor struct should be a slice")
		return nil
	}

	endpoints := make([]services.Endpoint, seField.Len(), seField.Len())
	for i := 0; i < seField.Len(); i++ {
		// Letting it blow up on errors gives an error message with information
		// that can't be easily gotten otherwise
		e := seField.Index(i).Addr().Interface().(services.Endpoint)

		e.EnsureID()
		endpoints[i] = e
	}

	return endpoints
}
