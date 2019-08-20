package haproxy

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/pointer"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheus/haproxy/prometheus"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"time"
)

// Config is the config for this monitor.
// Config implements ConfigInterface through prometheusexporter.Config.
type Config struct {
	prometheusexporter.Config                           `yaml:",inline" acceptsEndpoints:"true"`
	ExporterConfig                     *exporter.Config `yaml:"exporter"`
	UseLegacyCollectdPluginMetricNames bool             `yaml:"useLegacyCollectdPluginMetricNames"`
}

//// Validate k8s-specific configuration.
//func (c *Config) Validate() error {
//	return nil
//}

// NewDatapointSender is a ConfigInterface method implementation that creates a datapoint sender.
func (c *Config) NewDatapointSender() *prometheusexporter.DatapointSender {
	return &prometheusexporter.DatapointSender{
		SendDatapoints: func(output types.Output, dps []*datapoint.Datapoint) {
			now := time.Now()
			for i := range dps {
				dps[i].Timestamp = now
				if c.UseLegacyCollectdPluginMetricNames {
					renameDimensions(dps[i])
					addDimensions(dps[i])
					renameMetrics(dps[i])
				}
				output.SendDatapoint(dps[i])
			}
		},
	}
}

func renameDimensions(dp *datapoint.Datapoint) {
	if dims := dp.Dimensions; dims != nil {
		if dims["proxy"] != "" {
			dims["proxy_name"] = dims["proxy"]
			delete(dims, "proxy")
		}
		if dims["server"] != "" {
			dims["service_name"] = dims["server"]
			delete(dims, "server")
		}
	}
}

func addDimensions(dp *datapoint.Datapoint) {
	for k, v := range metricSet[dp.Metric].Dimensions {
		dp.Dimensions[k] = v
	}
}

func renameMetrics(dp *datapoint.Datapoint) {
	switch dp.Metric {
	case haproxyFrontendHTTPResponses, haproxyBackendHTTPResponses, haproxyServerHTTPResponses:
		switch dp.Dimensions["code"] {
		case   "1xx": dp.Metric = "derive.response_1xx"
		case   "2xx": dp.Metric = "derive.response_2xx"
		case   "3xx": dp.Metric = "derive.response_3xx"
		case   "4xx": dp.Metric = "derive.response_4xx"
		case   "5xx": dp.Metric = "derive.response_5xx"
		case "other": dp.Metric = "derive.response_other"
		}
	default: dp.Metric = string(aliases[Metric(dp.Metric)][aliasKeyCollectdPluginName])
	}
}

func (c *Config)SetExporterDefaults() {
	if c.ExporterConfig == nil { return }
	if c.ExporterConfig.ListenAddress  ==  "" { c.ExporterConfig.ListenAddress  = ":9101" }
	if c.ExporterConfig.MetricsPath    ==  "" { c.ExporterConfig.MetricsPath    = "/metrics" }
	if c.ExporterConfig.ScrapeURI      ==  "" { c.ExporterConfig.ScrapeURI      = "http://localhost/;csv" }
	if c.ExporterConfig.SSLVerify      == nil { c.ExporterConfig.SSLVerify      = pointer.Bool(true) }
	if c.ExporterConfig.TimeoutSeconds == nil { c.ExporterConfig.TimeoutSeconds = pointer.Int(5) }
}

// GetMonitorType is a ConfigInterface method implementation for getting the monitor type.
func (c *Config) GetMonitorType() string {
	return monitorType
}
