package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Collectd - struct for collectd configurations
type Collectd struct {
	Interval             *int  `yaml:"interval,omitempty"`
	Timeout              *int  `yaml:"timeout,omitempty"`
	ReadThreads          *int  `yaml:"readThreads,omitempty"`
	WriteQueueLimitHigh  *int  `yaml:"writeQueueLimitHigh,omitempty"`
	WriteQueueLimitLow   *int  `yaml:"writeQueueLimitLow,omitempty"`
	CollectInternalStats *bool `yaml:"collectInternalStats,omitempty"`
}

// LoadYaml - load yaml file
func (c *Collectd) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &c); err != nil {
			return err
		}
	}
	return err
}

// ParseConfig - parse configurations
func (c *Collectd) Parse(collectd map[string]interface{}) error {
	// Parse the interval used for collectd
	if c.Interval != nil {
		collectd["interval"] = *c.Interval
	}
	if c.Timeout != nil {
		collectd["timeout"] = *c.Timeout
	}
	if c.ReadThreads != nil {
		collectd["readThreads"] = *c.ReadThreads
	}
	if c.WriteQueueLimitHigh != nil {
		collectd["writeQueueLimitHigh"] = *c.WriteQueueLimitHigh
	}
	if c.WriteQueueLimitLow != nil {
		collectd["writeQueueLimitLow"] = *c.WriteQueueLimitLow
	}
	if c.CollectInternalStats != nil {
		collectd["collectInternalStats"] = *c.CollectInternalStats
	}
	return nil
}
