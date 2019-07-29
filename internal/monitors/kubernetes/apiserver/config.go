package apiserver

import (
	"io"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/signalfx/signalfx-agent/internal/core/common/kubernetes"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &prometheusexporter.Monitor{} }, &Config{})
}

// Config is the config for this monitor and implements ConfigInterface.
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Configuration of the Kubernetes API client.
	KubernetesAPI *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
	// Path to the metrics endpoint on server, usually `/metrics` (the default).
	MetricPath string `yaml:"metricPath" default:"/metrics"`
}

// Validate k8s-specific configuration.
func (c *Config) Validate() error {
	return c.KubernetesAPI.Validate()
}

// NewClient is a ConfigInterface method implementation that creates the prometheus client.
func (c *Config) NewClient() (*prometheusexporter.Client, error) {
	k8sClient, err := kubernetes.MakeClient(c.KubernetesAPI)
	if err != nil {
		return nil, err
	}
	return &prometheusexporter.Client{
		GetBodyReader: func() (bodyReader io.ReadCloser, format expfmt.Format, err error) {
			format = expfmt.FmtText
			bodyReader, err = k8sClient.CoreV1().RESTClient().Get().RequestURI(c.MetricPath).Stream()
			return
		},
	}, nil
}

// GetInterval is a ConfigInterface method implementation for getting the configured monitor run interval.
func (c *Config) GetInterval() time.Duration {
	return time.Duration(c.IntervalSeconds) * time.Second
}

// GetMonitorType is a ConfigInterface method implementation for getting the monitor type.
func (c *Config) GetMonitorType() string {
	return monitorType
}
