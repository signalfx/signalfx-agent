package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Proxy - stores proxy configurations
type Proxy struct {
	HTTP  string
	HTTPS string
	Skip  string
}

// LoadYaml - load yaml file
func (p *Proxy) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &p); err != nil {
			return err
		}
	}
	return err
}

// Parse -
func (p *Proxy) Parse(proxy map[string]string) error {
	proxy["http"] = p.HTTP
	proxy["https"] = p.HTTPS
	proxy["skip"] = p.Skip
	return nil
}
