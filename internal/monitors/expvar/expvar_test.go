package expvar

import (
	"github.com/signalfx/golib/datapoint"
	"testing"
)

type URLTest struct {
	useHTTPS bool
	host     string
	port     uint16
	path     string
	want     string
}

//var urlTests = []URLTest{
//	{useHTTPS: false, host: "localhost", port: 8080, path: "/debug/vars", want: "http://localhost:8080/debug/vars"},
//	{useHTTPS: false, host: "localhost", port: 8080, path: "debug/vars", want: "http://localhost:8080/debug/vars"},
//	{useHTTPS: true, host: "localhost", port: 8080, path: "/debug/vars", want: "https://localhost:8080/debug/vars"},
//}
//
//func TestSetURL(t *testing.T) {
//	for _, test := range urlTests {
//		conf := Config{UseHTTPS: test.useHTTPS, Host: test.host, Port: test.port, Path: test.path}
//		conf.setURL()
//		got := conf.url.String()
//		if got != test.want {
//			t.Errorf("Config(UseHTTPS: %t, Host: %s, Port: %d, Path: %s) = %s; want %s", test.useHTTPS, test.host, test.port, test.path, got, test.want)
//		}
//	}
//}

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
