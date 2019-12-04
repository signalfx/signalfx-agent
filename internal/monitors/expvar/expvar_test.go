package expvar

import (
	"testing"
)

type MetricsConfigTest struct {
	mConf *MetricConfig
	want  string
}

var MetricsTest = []MetricsConfigTest{
	{mConf: &MetricConfig{JSONPath: "System.Cpu", PathSeparator: ".", Type: "gauge"}, want: "system.cpu"},
	{mConf: &MetricConfig{JSONPath: "System.Cpu[0]", PathSeparator: ".", Type: "gauge"}, want: "system.cpu[0]"},
	{mConf: &MetricConfig{JSONPath: "System.Cpu[0].CacheGCCPUFraction", PathSeparator: ".", Type: "gauge"}, want: "system.cpu[0].cache_gccpu_fraction"},
	{mConf: &MetricConfig{JSONPath: "System.Cpu.CacheGCCPUFraction", PathSeparator: ".", Type: "gauge"}, want: "system.cpu.cache_gccpu_fraction"},
}

func TestSetMetrics(t *testing.T) {
	for _, test := range MetricsTest {
		test.mConf.setName()
		got := test.mConf.Name
		if got != test.want {
			t.Errorf("got metric name: %s, want metric name: %s", got, test.want)
		}
	}
}
