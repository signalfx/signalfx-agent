package config

// ObserverConfig holds the configuration for an observer
type ObserverConfig struct {
	// The type of the observer
	Type        string                 `yaml:"type,omitempty"`
	OtherConfig map[string]interface{} `yaml:",inline" default:"{}"`
}

// ExtraConfig returns generic config as a map
func (oc *ObserverConfig) ExtraConfig() map[string]interface{} {
	return oc.OtherConfig
}
