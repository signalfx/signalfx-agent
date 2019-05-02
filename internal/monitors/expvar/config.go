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
	MetricConfigs []*MetricConfig `yaml:"metrics"`
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
	DimensionConfigs []*DimensionConfig `yaml:"dimensions"`

	metricType datapoint.MetricType
}

// DimensionConfig for metric dimension configuration
type DimensionConfig struct {
	// Dimension name
	Name string `yaml:"name" validate:"required"`
	// JSON path of the dimension value
	JSONPath string `yaml:"JSONPath"`
	// Dimension value
	Value string `yaml:"value"`
}

// Validate validates configuration
func (conf *Config) Validate() error {
	if conf.MetricConfigs != nil {
		for _, mConf := range conf.MetricConfigs {
			if err := validation.ValidateStruct(mConf); err != nil {
				return err
			}
			switch mConf.Type {
			case cumulative:
				mConf.metricType = datapoint.Counter
			case gauge:
				mConf.metricType = datapoint.Gauge
			default:
				panic(fmt.Sprintf("unexpected metric type %s", mConf.Type))
			}

			// Validating dimension configuration
			for _, dConf := range mConf.DimensionConfigs {
				if err := validation.ValidateStruct(dConf); err != nil {
					return err
				}
				switch {
				// Not sure when this is nil.
				case dConf == nil:
					continue
				case strings.ContainsRune(dConf.Name, ' '):
					return fmt.Errorf("dimension name cannot have whitespaces")
				case dConf.JSONPath != "" && dConf.Value != "":
					return fmt.Errorf("dimension path %s and dimension value %s cannot be configure simultaneously", dConf.JSONPath, dConf.Value)
				case dConf.JSONPath != "" && !strings.HasPrefix(mConf.JSONPath, dConf.JSONPath):
					return fmt.Errorf("dimension path %s must be shorter than metric path %s and start from the same root", dConf.JSONPath, mConf.JSONPath)
				}
			}
		}
	}
	return nil
}

func (mConf *MetricConfig) name() string {
	if strings.TrimSpace(mConf.Name) == "" {
		return toSnakeCase(mConf.JSONPath)
	}
	return mConf.Name
}
