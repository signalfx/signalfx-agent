package expvar

import (
	"testing"
)

type URLTest struct {
	useHTTPS bool
	host     string
	port     uint16
	path     string
	want     string
}

var urlTests = []URLTest{
	{useHTTPS: false, host: "localhost", port: 8080, path: "/debug/vars", want: "http://localhost:8080/debug/vars"},
	{useHTTPS: false, host: "localhost", port: 8080, path: "debug/vars", want: "http://localhost:8080/debug/vars"},
	{useHTTPS: true, host: "localhost", port: 8080, path: "/debug/vars", want: "https://localhost:8080/debug/vars"},
}

func TestSetURL(t *testing.T) {
	for _, test := range urlTests {
		conf := Config{UseHTTPS: test.useHTTPS, Host: test.host, Port: test.port, Path: test.path}
		conf.setURL()
		got := conf.url.String()
		if got != test.want {
			t.Errorf("Config(UseHTTPS: %t, Host: %s, Port: %d, Path: %s) = %s; want %s", test.useHTTPS, test.host, test.port, test.path, got, test.want)
		}
	}
}

type MetricsConfigTest struct {
	conf *Config
	want *metric
}

var MetricsTest = []MetricsConfigTest{
	{&Config{Metrics: []*metric{{JSONPath: "System.Cpu", Type: gauge}}}, &metric{JSONPath: "System.Cpu", Name: "system.cpu", Type: gauge}},
	{&Config{Metrics: []*metric{{JSONPath: "System.Cpu[0]", Type: gauge}}}, &metric{JSONPath: "System.Cpu[0]", Name: "system.cpu[0]", Type: gauge}},
	{&Config{Metrics: []*metric{{JSONPath: "System.Cpu[0].CacheGCCPUFraction", Type: gauge}}}, &metric{JSONPath: "System.Cpu[0].CacheGCCPUFraction", Name: "system.cpu[0].cache_gccpu_fraction", Type: gauge}},
	{&Config{Metrics: []*metric{{JSONPath: "System.Cpu.CacheGCCPUFraction", Type: gauge}}}, &metric{JSONPath: "System.Cpu.CacheGCCPUFraction", Name: "system.cpu.cache_gccpu_fraction", Type: gauge}},
}

func TestSetMetrics(t *testing.T) {
	for _, test := range MetricsTest {
		test.conf.initMetrics()
		var got *metric
		for _, got = range test.conf.Metrics {
			if (*got).JSONPath == (*test.want).JSONPath {
				if (*got).Name != (*test.want).Name {
					t.Errorf("got metric name: %s, want metric name: %s", (*got).Name, (*test.want).Name)
				}
			}
		}
	}
}
