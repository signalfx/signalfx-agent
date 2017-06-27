package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// TLS stores tls configurations
type TLS struct {
	SkipVerify bool   `yaml:"skipVerify"`
	ClientCert string `yaml:"clientCert"`
	ClientKey  string `yaml:"clientKey"`
	CACert     string `yaml:"caCert"`
}

// LoadYAML loads a yaml file
func (t *TLS) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &t); err != nil {
			return err
		}
	}
	return err
}

// Parse parses the configuration into a supplied map
func (t *TLS) Parse(tls map[string]interface{}) error {
	tls["caCert"] = t.CACert
	tls["skipVerify"] = t.SkipVerify
	tls["clientCert"] = t.ClientCert
	tls["clientKey"] = t.ClientKey
	return nil
}
