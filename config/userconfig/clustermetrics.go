package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// ClusterMetrics struct for storing cluster metric configurations
type ClusterMetrics struct {
	IsClusterReporter  *bool    `yaml:"alwaysClusterReporter,omitempty"`
	NamespaceFilter   []string `yaml:"namespaceFilter,omitempty"`
	MetricFilter      []string `yaml:"metricFilter,omitempty"`
	IntervalSeconds    *int     `yaml:"intervalSeconds,omitempty"`
}

// LoadYAML loads a yaml file
func (c *ClusterMetrics) LoadYAML(path string) error {
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

// Parse aggregates cluster metric configurations
func (c *ClusterMetrics) Parse(clusterMetrics map[string]interface{}) error {
	if c.IsClusterReporter != nil {
		clusterMetrics["alwaysClusterReporter"] = *c.IsClusterReporter
	}
	if len(c.MetricFilter) >= 0 {
		clusterMetrics["metricFilter"] = c.MetricFilter
	}
	if len(c.NamespaceFilter) >= 0 {
		clusterMetrics["namespaceFilter"] = c.NamespaceFilter
	}
	if c.IntervalSeconds != nil {
		clusterMetrics["intervalSeconds"] = *c.IntervalSeconds
	}
	return nil
}
