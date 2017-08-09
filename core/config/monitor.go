package config

import (
	"fmt"
	"net/url"

	"github.com/signalfx/neo-agent/core/config/stores"
	"github.com/signalfx/neo-agent/core/filters"
)

// MonitorID is a unique id for monitors
type MonitorID string

// MonitorConfig configures a given monitor instance.  There is a 1-1
// correspondance between monitor config and monitor instances.  There will
// never be more monitor instances than there are monitor configurations, since
// all services that match the same discovery rule will be added to the same
// monitor instance.  If a monitor's discovery rule does not match any
// discovered services, the monitor will not run.
type MonitorConfig struct {
	Type string `yaml:"type,omitempty"`
	// Id can be used to uniquely identify monitors so that they can be
	// reconfigured in place instead of destroyed and recreated
	Id            MonitorID `yaml:"id,omitempty"`
	DiscoveryRule string    `yaml:"discoveryRule,omitempty"`
	// Extra dimensions specific to this monitor
	ExtraDimensions map[string]string `yaml:"extraDimensions,omitempty" default:"{}"`
	// K8s pod label keys to send as dimensions
	K8sLabelDimensions []string `yaml:"labelDimensions,omitempty" default:"[]"`
	// If unset or 0, will default to the top-level IntervalSeconds value
	IntervalSeconds int                    `yaml:"intervalSeconds,omitempty" default:"0"`
	OtherConfig     map[string]interface{} `yaml:",inline" default:"{}" json:"-"`
	// The remaining are propagated from the top-level config and cannot be set
	// by the user directly on the monitor
	IngestURL           *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	Hostname            string             `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
	// Most monitors can ignore this
	GlobalDimensions map[string]string `yaml:"-" default:"{}"`
	ProcFSPath       string            `yaml:"-"`
	MetaStore        *stores.MetaStore `yaml:"-"`
}

func (mc *MonitorConfig) GetOtherConfig() map[string]interface{} {
	return mc.OtherConfig
}

func (mc *MonitorConfig) EnsureID(generator func(string) int) {
	if len(mc.Id) == 0 {
		mc.Id = MonitorID(fmt.Sprintf("%s-%d", mc.Type, generator(mc.Type)))
	}
}
