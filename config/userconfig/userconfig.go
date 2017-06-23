package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// UserConfig - top level user configuration struct
type UserConfig struct {
	Collectd   *Collectd `yaml:"collectd,omitempty"`
	Filter     *Filter   `yaml:"filterContianerMetrics,omitempty"`
	IngestURL  string    `yaml:"ingestURL,omitempty"`
	Kubernetes *Kubernetes
	Mesosphere *Mesosphere
	Proxy      *Proxy
}

// LoadYaml - load yaml file
func (u *UserConfig) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &u); err != nil {
			return err
		}
	}
	return err
}
