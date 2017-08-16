package config

type ObserverConfig struct {
	Type string `yaml:"type,omitempty"`
	// Id can be used to uniquely identify observers so that they can be
	// reconfigured in place instead of destroyed and recreated
	Id          string                 `yaml:"id,omitempty"`
	OtherConfig map[string]interface{} `yaml:",inline" default:"{}"`
}

func (oc *ObserverConfig) GetOtherConfig() map[string]interface{} {
	return oc.OtherConfig
}
