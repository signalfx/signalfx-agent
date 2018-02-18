package config

// ObserverConfig holds the configuration for an observer
type ObserverConfig struct {
	// The type of the observer
	Type        string                 `yaml:"type,omitempty"`
	OtherConfig map[string]interface{} `yaml:",inline" default:"{}"`
	// The following are propagated down from the main config and cannot be set
	// by the user on the observer config.
	Hostname string `yaml:"-"`
}

// ExtraConfig returns generic config as a map
func (oc *ObserverConfig) ExtraConfig() map[string]interface{} {
	return oc.OtherConfig
}
