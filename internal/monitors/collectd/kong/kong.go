// +build !windows

package kong

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/kong"

// MONITOR(collectd/kong): Monitors a Kong instance using [collectd-kong](https://github.com/signalfx/collectd-kong).
//
// See the [integration documentation](https://github.com/signalfx/integrations/tree/master/collectd-kong)
// for more information.
//
// The `metrics` field below is populated with a set of metrics that are
// described at https://github.com/signalfx/collectd-kong/blob/master/README.md.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//   - type: collectd/kong
//     host: 127.0.0.1
//     port: 8001
//     metrics:
//       - metric: request_latency
//         report: true
//       - metric: connections_accepted
//         report: false
// ```
//
// Sample YAML configuration with custom /signalfx route and white and blacklists
//
// ```yaml
// monitors:
//   - type: collectd/kong
//     host: 127.0.0.1
//     port: 8443
//     url: https://127.0.0.1:8443/routed_signalfx
//     authHeader:
//       header: Authorization
//       value: HeaderValue
//     metrics:
//       - metric: request_latency
//         report: true
//     reportStatusCodeGroups: true
//     statusCodes:
//       - 202
//       - 403
//       - 405
//       - 419
//       - "5*"
//     serviceNamesBlacklist:
//       - "*SomeService*"
// ```
//

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.Monitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Header defines name/value pair for AuthHeader option
type Header struct {
	// Name of header to include with GET
	HeaderName string `yaml:"header" validate:"required"`
	// Value of header
	Value string `yaml:"value" validate:"required"`
}

// Metric is for use with `Metric "metric_name" bool` collectd plugin format
type Metric struct {
	// Name of metric, per collectd-kong
	MetricName string `yaml:"metric" validate:"required"`
	// Whether to report this metric
	ReportBool bool `yaml:"report" validate:"required"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	python.CoreConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Kong host to connect with (used for autodiscovery and URL)
	Host string `yaml:"host" validate:"required"`
	// Port for kong-plugin-signalfx hosting server (used for autodiscovery and URL)
	Port uint16 `yaml:"port" validate:"required"`
	// Registration name when using multiple instances in Smart Agent
	Name string `yaml:"name"`
	// kong-plugin-signalfx metric plugin
	URL string `yaml:"url" default:"http://{{.Host}}:{{.Port}}/signalfx"`
	// Header and its value to use for requests to SFx metric endpoint
	AuthHeader *Header `yaml:"authHeader"`
	// Whether to verify certificates when using ssl/tls
	VerifyCerts *bool `yaml:"verifyCerts"`
	// CA Bundle file or directory
	CABundle string `yaml:"caBundle"`
	// Client certificate file (with or without included key)
	ClientCert string `yaml:"clientCert"`
	// Client cert key if not bundled with clientCert
	ClientCertKey string `yaml:"clientCertKey"`
	// Whether to use debug logging for collectd-kong
	Verbose *bool `yaml:"verbose"`
	// List of metric names and report flags. See monitor description for more
	// details.
	Metrics []Metric `yaml:"metrics"`
	// Report metrics for distinct API IDs where applicable
	ReportAPIIDs *bool `yaml:"reportApiIds"`
	// Report metrics for distinct API names where applicable
	ReportAPINames *bool `yaml:"reportApiNames"`
	// Report metrics for distinct Service IDs where applicable
	ReportServiceIDs *bool `yaml:"reportServiceIds"`
	// Report metrics for distinct Service names where applicable
	ReportServiceNames *bool `yaml:"reportServiceNames"`
	// Report metrics for distinct Route IDs where applicable
	ReportRouteIDs *bool `yaml:"reportRouteIds"`
	// Report metrics for distinct HTTP methods where applicable
	ReportHTTPMethods *bool `yaml:"reportHttpMethods"`
	// Report metrics for distinct HTTP status code groups (eg. "5xx") where applicable
	ReportStatusCodeGroups *bool `yaml:"reportStatusCodeGroups"`
	// Report metrics for distinct HTTP status codes where applicable (mutually exclusive with ReportStatusCodeGroups)
	ReportStatusCodes *bool `yaml:"reportStatusCodes"`

	// List of API ID patterns to report distinct metrics for, if reportApiIds is false
	APIIDs []string `yaml:"apiIds"`
	// List of API ID patterns to not report distinct metrics for, if reportApiIds is true or apiIds are specified
	APIIDsBlacklist []string `yaml:"apiIdsBlacklist"`
	// List of API name patterns to report distinct metrics for, if reportApiNames is false
	APINames []string `yaml:"apiNames"`
	// List of API name patterns to not report distinct metrics for, if reportApiNames is true or apiNames are specified
	APINamesBlacklist []string `yaml:"apiNamesBlacklist"`
	// List of Service ID patterns to report distinct metrics for, if reportServiceIds is false
	ServiceIDs []string `yaml:"serviceIds"`
	// List of Service ID patterns to not report distinct metrics for, if reportServiceIds is true or serviceIds are specified
	ServiceIDsBlacklist []string `yaml:"serviceIdsBlacklist"`
	// List of Service name patterns to report distinct metrics for, if reportServiceNames is false
	ServiceNames []string `yaml:"serviceNames"`
	// List of Service name patterns to not report distinct metrics for, if reportServiceNames is true or serviceNames are specified
	ServiceNamesBlacklist []string `yaml:"serviceNamesBlacklist"`
	// List of Route ID patterns to report distinct metrics for, if reportRouteIds is false
	RouteIDs []string `yaml:"routeIds"`
	// List of Route ID patterns to not report distinct metrics for, if reportRouteIds is true or routeIds are specified
	RouteIDsBlacklist []string `yaml:"routeIdsBlacklist"`
	// List of HTTP method patterns to report distinct metrics for, if reportHttpMethods is false
	HTTPMethods []string `yaml:"httpMethods"`
	// List of HTTP method patterns to not report distinct metrics for, if reportHttpMethods is true or httpMethods are specified
	HTTPMethodsBlacklist []string `yaml:"httpMethodsBlacklist"`
	// List of HTTP status code patterns to report distinct metrics for, if reportStatusCodes is false
	StatusCodes []string `yaml:"statusCodes"`
	// List of HTTP status code patterns to not report distinct metrics for, if reportStatusCodes is true or statusCodes are specified
	StatusCodesBlacklist []string `yaml:"statusCodesBlacklist"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.PluginConfig = map[string]interface{}{
		"URL": conf.URL,
	}
	if conf.ModuleName == "" {
		conf.ModuleName = "kong_plugin"
	}
	if len(conf.ModulePaths) == 0 {
		conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "kong")}
	}
	if len(conf.TypesDBPaths) == 0 {
		conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	}

	if conf.Verbose != nil {
		conf.PluginConfig["Verbose"] = *conf.Verbose
	}
	if conf.Name != "" {
		conf.PluginConfig["Name"] = conf.Name
	}
	if conf.AuthHeader != nil {
		conf.PluginConfig["AuthHeader"] = []string{conf.AuthHeader.HeaderName, conf.AuthHeader.Value}
	}
	if conf.VerifyCerts != nil {
		conf.PluginConfig["VerifyCerts"] = *conf.VerifyCerts
	}
	if conf.CABundle != "" {
		conf.PluginConfig["CABundle"] = conf.CABundle
	}
	if conf.ClientCert != "" {
		conf.PluginConfig["ClientCert"] = conf.ClientCert
	}
	if conf.ClientCertKey != "" {
		conf.PluginConfig["ClientCertKey"] = conf.ClientCertKey
	}
	if conf.IntervalSeconds > 0 {
		conf.PluginConfig["Interval"] = conf.IntervalSeconds
	}
	if len(conf.Metrics) > 0 {
		values := make([][]interface{}, 0, len(conf.Metrics))
		for _, m := range conf.Metrics {
			values = append(values, []interface{}{m.MetricName, m.ReportBool})
		}
		conf.PluginConfig["Metric"] = map[string]interface{}{
			"#flatten": true,
			"values":   values,
		}
	}
	if conf.ReportAPIIDs != nil {
		conf.PluginConfig["ReportAPIIDs"] = *conf.ReportAPIIDs
	}
	if conf.ReportAPINames != nil {
		conf.PluginConfig["ReportAPINames"] = *conf.ReportAPINames
	}
	if conf.ReportServiceIDs != nil {
		conf.PluginConfig["ReportServiceIDs"] = *conf.ReportServiceIDs
	}
	if conf.ReportServiceNames != nil {
		conf.PluginConfig["ReportServiceNames"] = *conf.ReportServiceNames
	}
	if conf.ReportRouteIDs != nil {
		conf.PluginConfig["ReportRouteIDs"] = *conf.ReportRouteIDs
	}
	if conf.ReportHTTPMethods != nil {
		conf.PluginConfig["ReportHTTPMethods"] = *conf.ReportHTTPMethods
	}
	if conf.ReportStatusCodeGroups != nil {
		conf.PluginConfig["ReportStatusCodeGroups"] = *conf.ReportStatusCodeGroups
	}
	if conf.ReportStatusCodes != nil {
		conf.PluginConfig["ReportStatusCodes"] = *conf.ReportStatusCodes
	}
	if len(conf.APIIDs) > 0 {
		conf.PluginConfig["APIIDs"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.APIIDs,
		}
	}
	if len(conf.APIIDsBlacklist) > 0 {
		conf.PluginConfig["APIIDsBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.APIIDsBlacklist,
		}
	}
	if len(conf.APINames) > 0 {
		conf.PluginConfig["APINames"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.APINames,
		}
	}
	if len(conf.APINamesBlacklist) > 0 {
		conf.PluginConfig["APINamesBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.APINamesBlacklist,
		}
	}
	if len(conf.ServiceIDs) > 0 {
		conf.PluginConfig["ServiceIDs"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ServiceIDs,
		}
	}
	if len(conf.ServiceIDsBlacklist) > 0 {
		conf.PluginConfig["ServiceIDsBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ServiceIDsBlacklist,
		}
	}
	if len(conf.ServiceNames) > 0 {
		conf.PluginConfig["ServiceNames"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ServiceNames,
		}
	}
	if len(conf.ServiceNamesBlacklist) > 0 {
		conf.PluginConfig["ServiceNamesBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.ServiceNamesBlacklist,
		}
	}
	if len(conf.RouteIDs) > 0 {
		conf.PluginConfig["RouteIDs"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.RouteIDs,
		}
	}
	if len(conf.RouteIDsBlacklist) > 0 {
		conf.PluginConfig["RouteIDsBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.RouteIDsBlacklist,
		}
	}
	if len(conf.HTTPMethods) > 0 {
		conf.PluginConfig["HTTPMethods"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.HTTPMethods,
		}
	}
	if len(conf.HTTPMethodsBlacklist) > 0 {
		conf.PluginConfig["HTTPMethodsBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.HTTPMethodsBlacklist,
		}
	}
	if len(conf.StatusCodes) > 0 {
		conf.PluginConfig["StatusCodes"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.StatusCodes,
		}
	}
	if len(conf.StatusCodesBlacklist) > 0 {
		conf.PluginConfig["StatusCodesBlacklist"] = map[string]interface{}{
			"#flatten": true,
			"values":   conf.StatusCodesBlacklist,
		}
	}

	return m.Monitor.Configure(conf)
}
