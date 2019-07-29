package prometheusexporter

import (
	"io"
	"time"

	"github.com/prometheus/common/expfmt"
)

// ConfigInterface is the interface for configuring the prometheus exporter monitor.
type ConfigInterface interface {
	NewClient() (*Client, error)
	GetInterval() time.Duration
	GetMonitorType() string
}

// Client is the prometheus exporter monitor client for reading prometheus metrics.
type Client struct {
	GetBodyReader func() (bodyReader io.ReadCloser, format expfmt.Format, err error)
}
