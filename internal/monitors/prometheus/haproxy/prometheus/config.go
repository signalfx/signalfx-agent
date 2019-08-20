package exporter

type Config struct {
	// Address to listen on for web interface and telemetry.
	ListenAddress      string `yaml:"listenAddress" default:":9101"`
	// Path under which to expose metrics.
	MetricsPath        string `yaml:"metricPath" default:"/metrics"`
	// URI on which to scrape HAProxy.
	ScrapeURI          string `yaml:"scrapeURI" default:"http://localhost/;csv"`
	// Flag that enables SSL certificate verification for the scrape URI.
	SSLVerify          *bool  `yaml:"sslVerify" default:"true"`
	// Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1
	ServerMetricFields string `yaml:"serverMetricFields"`
	// Timeout for trying to get stats from HAProxy.
	TimeoutSeconds     *int   `yaml:"timeoutSeconds", default:"5"`
	// Path to HAProxy pid file.
	//
	// If provided, the standard process metrics get exported for the HAProxy process, prefixed with
	// 'haproxy_process_...'. The haproxy_process exporter needs to have read access to files owned by the
	// HAProxy process. Depends on the availability of /proc.
	//
	// https://prometheus.io/docs/instrumenting/writing_clientlibs/#process-metrics.
	PidFile            string `yaml:"pidFile"`
}
