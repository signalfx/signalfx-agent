package expvar

import (
	"fmt"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/core/config/validation"

	"github.com/signalfx/golib/datapoint"

	"github.com/signalfx/signalfx-agent/internal/core/config"
)

// Config for monitor configuration
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Host of the expvar endpoint
	Host string `yaml:"host" validate:"required"`
	// Port of the expvar endpoint
	Port uint16 `yaml:"port" validate:"required"`
	// If true, the agent will connect to the host using HTTPS instead of plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`
	// If useHTTPS is true and this option is also true, the host's TLS cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`
	// Path to the expvar endpoint, usually `/debug/vars` (the default).
	Path string `yaml:"path" default:"/debug/vars"`
	// If true, sends metrics memstats.alloc, memstats.by_size.size, memstats.by_size.mallocs and memstats.by_size.frees
	EnhancedMetrics bool `yaml:"enhancedMetrics"`
	// Metrics configurations
	MetricConfigs []MetricConfig `yaml:"metrics"`
}

// MetricConfig for metric configuration
type MetricConfig struct {
	// Metric name
	Name string `yaml:"name"`
	// JSON path of the metric value
	JSONPath string `yaml:"JSONPath" validate:"required"`
	// SignalFx metric type. Possible values are "gauge" or "cumulative"
	Type string `yaml:"type" validate:"required,oneof=gauge cumulative"`
	// Metric dimensions
	DimensionConfigs []DimensionConfig `yaml:"dimensions"`
}

func (mc *MetricConfig) metricType() datapoint.MetricType {
	switch mc.Type {
	case "cumulative":
		return datapoint.Counter
	default:
		return datapoint.Gauge
	}
}

// DimensionConfig for metric dimension configuration
type DimensionConfig struct {
	// Dimension name
	Name string `yaml:"name" validate:"required,excludes=0x20"`
	// JSON path of the dimension value
	JSONPath string `yaml:"JSONPath"`
	// Dimension value
	Value string `yaml:"value"`
}

// Validate validates configuration
func (c *Config) Validate() error {
	if c.MetricConfigs != nil {
		for _, mConf := range c.MetricConfigs {
			if err := validation.ValidateStruct(mConf); err != nil {
				return err
			}

			// Validating dimension configuration
			for _, dConf := range mConf.DimensionConfigs {
				switch {
				case dConf.JSONPath != "" && dConf.Value != "":
					return fmt.Errorf("dimension path %s and dimension value %s cannot be configured simultaneously", dConf.JSONPath, dConf.Value)
				case dConf.JSONPath != "" && !strings.HasPrefix(mConf.JSONPath, dConf.JSONPath):
					return fmt.Errorf("dimension path %s must be shorter than metric path %s and start from the same root", dConf.JSONPath, mConf.JSONPath)
				}
			}
		}
	}
	return nil
}

func (c *Config) getAllMetricConfigs() []MetricConfig {
	configs := append([]MetricConfig{}, c.MetricConfigs...)

	memstatsMetricPathsGauge := []string{
		"memstats.HeapAlloc", "memstats.HeapIdle", "memstats.HeapInuse", "memstats.HeapReleased",
		"memstats.HeapObjects", "memstats.StackInuse", "memstats.StackSys", "memstats.MSpanInuse", "memstats.MSpanSys",
		"memstats.MCacheInuse", "memstats.MCacheSys", "memstats.BuckHashSys", "memstats.GCSys", "memstats.OtherSys",
		"memstats.Sys", "memstats.NextGC", "memstats.LastGC", "memstats.GCCPUFraction", "memstats.EnableGC",
		memstatsPauseNsMetricPath, memstatsPauseEndMetricPath,
	}
	memstatsMetricPathsCumulative := []string{
		"memstats.TotalAlloc", "memstats.Lookups", "memstats.Mallocs", "memstats.Frees", "memstats.PauseTotalNs",
		memstatsNumGCMetricPath, "memstats.NumForcedGC",
	}

	if c.EnhancedMetrics {
		memstatsMetricPathsGauge = append(memstatsMetricPathsGauge, "memstats.HeapSys", "memstats.DebugGC", "memstats.Alloc")
		memstatsMetricPathsCumulative = append(memstatsMetricPathsCumulative, memstatsBySizeSizeMetricPath, memstatsBySizeMallocsMetricPath, memstatsBySizeFreesMetricPath)
	}
	for _, path := range memstatsMetricPathsGauge {
		configs = append(configs, MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: "gauge", DimensionConfigs: []DimensionConfig{{}}})
	}
	for _, path := range memstatsMetricPathsCumulative {
		configs = append(configs, MetricConfig{Name: toSnakeCase(path), JSONPath: path, Type: "cumulative", DimensionConfigs: []DimensionConfig{{}}})
	}

	return configs
}

func (mc *MetricConfig) name() string {
	if strings.TrimSpace(mc.Name) == "" {
		return toSnakeCase(mc.JSONPath)
	}
	return mc.Name
}
