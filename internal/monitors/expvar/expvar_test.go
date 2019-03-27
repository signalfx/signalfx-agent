package expvar

import (
	"github.com/signalfx/golib/datapoint"
	"testing"
)

type MetricsConfigTest struct {
	monitor *Monitor
	conf *Config
	want *MetricConfig
}


var MetricsTest = []MetricsConfigTest{
	{monitor: &Monitor{metricTypes: map[*MetricConfig]datapoint.MetricType{}, metricPathsParts: map[*MetricConfig][]string{}, dimensionPathsParts: map[*DimensionConfig][]string{}}, conf: &Config{MetricConfigs: []*MetricConfig{{JSONPath: "System.Cpu", Type: gauge}}}, want: &MetricConfig{JSONPath: "System.Cpu", Name: "system.cpu", Type: gauge}},
	{monitor: &Monitor{metricTypes: map[*MetricConfig]datapoint.MetricType{}, metricPathsParts: map[*MetricConfig][]string{}, dimensionPathsParts: map[*DimensionConfig][]string{}}, conf: &Config{MetricConfigs: []*MetricConfig{{JSONPath: "System.Cpu[0]", Type: gauge}}}, want: &MetricConfig{JSONPath: "System.Cpu[0]", Name: "system.cpu[0]", Type: gauge}},
	{monitor: &Monitor{metricTypes: map[*MetricConfig]datapoint.MetricType{}, metricPathsParts: map[*MetricConfig][]string{}, dimensionPathsParts: map[*DimensionConfig][]string{}}, conf: &Config{MetricConfigs: []*MetricConfig{{JSONPath: "System.Cpu[0].CacheGCCPUFraction", Type: gauge}}}, want: &MetricConfig{JSONPath: "System.Cpu[0].CacheGCCPUFraction", Name: "system.cpu[0].cache_gccpu_fraction", Type: gauge}},
	{monitor: &Monitor{metricTypes: map[*MetricConfig]datapoint.MetricType{}, metricPathsParts: map[*MetricConfig][]string{}, dimensionPathsParts: map[*DimensionConfig][]string{}}, conf: &Config{MetricConfigs: []*MetricConfig{{JSONPath: "System.Cpu.CacheGCCPUFraction", Type: gauge}}}, want: &MetricConfig{JSONPath: "System.Cpu.CacheGCCPUFraction", Name: "system.cpu.cache_gccpu_fraction", Type: gauge}},
}

func TestSetMetrics(t *testing.T) {
	for _, test := range MetricsTest {
		test.monitor.initMetrics(test.conf)
		var got *MetricConfig
		for _, got = range test.conf.MetricConfigs {
			if (*got).JSONPath == (*test.want).JSONPath {
				if (*got).Name != (*test.want).Name {
					t.Errorf("got metric name: %s, want metric name: %s", (*got).Name, (*test.want).Name)
				}
			}
		}
	}
}
