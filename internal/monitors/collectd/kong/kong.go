// +build !windows

package kong

//go:generate collectd-template-to-go kong.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/kong"

// MONITOR(collectd/kong): Monitors a Kong instance using [collectd-kong](https://github.com/signalfx/collectd-kong).
//
// See the [integration documentation](https://github.com/signalfx/integrations/tree/master/collectd-kong)
// for more information.
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
			*collectd.NewMonitorCore(CollectdTemplate),
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
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
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
	// List of metric names and report flags. For metric names see [collectd-kong](https://github.com/signalfx/collectd-kong/blob/master/README.md)
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
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (rm *Monitor) Configure(conf *Config) error {
	return rm.SetConfigurationAndRun(conf)
}
