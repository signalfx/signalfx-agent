package parse

import (
	"fmt"
)

// GetMetricName combines telegraf fields and tags into a full metric name
func GetMetricName(metric string, field string, dims map[string]string) string {
	var name = metric

	// If sf_prefix is provided
	if _, ok := dims["sf_prefix"]; ok {
		name = dims["sf_prefix"]
	}

	// Include field when it adds to the metric name
	if field != "value" {
		name = name + "." + field
	}

	// If sf_metric is provided
	if _, ok := dims["sf_metric"]; ok {
		// If sf_metric is provided
		name = dims["sf_metric"]
	}

	return name
}

// ModifyDimensions of the metric according to the following rules
func ModifyDimensions(name string, dims map[string]string, props map[string]interface{}) (err error) {
	// If the plugin doesn't define a plugin name use metric.Name()
	if _, in := dims["plugin"]; !in {
		dims["plugin"] = name
	}

	// remove sf_prefix if it exists in the dimension map
	if _, in := dims["sf_prefix"]; in {
		delete(dims, "sf_prefix")
	}

	// if sfMetric exists
	if sfMetric, in := dims["sf_metric"]; in {
		// if the metric is a metadata object
		if sfMetric == "objects.host-meta-data" {
			// If property exists remap it
			if _, in := dims["property"]; in {
				props["property"] = dims["property"]
				delete(dims, "property")
			} else {
				// This is a malformed metadata event
				err = fmt.Errorf("E! objects.host-metadata object doesn't have a property")
			}
			// remove the sf_metric dimension
			delete(dims, "sf_metric")
		}
	}
	return
}
