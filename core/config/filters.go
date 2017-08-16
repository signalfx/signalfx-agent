package config

import (
	"reflect"

	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// MetricFiltering describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	// Can map to either a []string or simple string
	Dimensions  map[string]interface{} `yaml:"dimensions,omitempty" default:"{}"`
	MetricNames []string               `yaml:"metricNames,omitempty" default:"[]"`
	MetricName  string                 `yaml:"metricName,omitempty"`
	MonitorType string                 `yaml:"monitorType,omitempty"`
}

func (mf *MetricFilter) ConvertDimensionsMapForSliceValues() map[string][]string {
	dims := make(map[string][]string)
	for k, d := range mf.Dimensions {
		if s, ok := d.(string); ok {
			dims[k] = []string{s}
		} else if interfaceSlice, ok := d.([]interface{}); ok {
			ss := utils.InterfaceSliceToStringSlice(interfaceSlice)
			if ss != nil {
				dims[k] = ss
			}
		}

		if dims[k] == nil {
			log.WithFields(log.Fields{
				"dimensionFilter": k,
				"value":           d,
				"type":            reflect.ValueOf(d).Type(),
			}).Error("Invalid dimension filter")
			return nil
		}
	}
	return dims
}

func (mf *MetricFilter) ConvertMetricNameToSlice() {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
}
