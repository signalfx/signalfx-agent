package httpclient

import (
	"crypto/tls"
	"github.com/signalfx/signalfx-agent/pkg/core/common/auth"
	"net/http"
	"time"
)

// HTTPConfig can be embedded inside a monitor config.
type HTTPConfig struct {
	// HTTP timeout duration for both read and writes. This should be a
	// duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	HTTPTimeout time.Duration `yaml:"httpTimeout" default:"10s"`

	// Basic Auth username to use on each request, if any.
	Username string `yaml:"username"`
	// Basic Auth password to use on each request, if any.
	Password string `yaml:"password" neverLog:"true"`

	// If true, the agent will connect to the exporter using HTTPS instead of plain HTTP.
	UseHTTPS bool `yaml:"useHTTPS"`

	// If useHTTPS is true and this option is also true, the exporter's TLS
	// cert will not be verified.
	SkipVerify bool `yaml:"skipVerify"`

	// Path to the CA cert that has signed the TLS cert, unnecessary
	// if `skipVerify` is set to false.
	CACertPath string `yaml:"caCertPath"`
	// Path to the client TLS cert to use for TLS required connections
	ClientCertPath string `yaml:"clientCertPath"`
	// Path to the client TLS key to use for TLS required connections
	ClientKeyPath string `yaml:"clientKeyPath"`
}

// Scheme returns https if enabled, otherwise http
func (h *HTTPConfig) Scheme() string {
	if h.UseHTTPS {
		return "https"
	}
	return "http"
}

// Build returns a configured http.Client
func (h *HTTPConfig) Build() (*http.Client, error) {
	roundTripper, err := func() (http.RoundTripper, error) {
		transport := http.DefaultTransport.(*http.Transport).Clone()

		if h.UseHTTPS {
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: h.SkipVerify,
			}
			if _, err := auth.TLSConfig(transport.TLSClientConfig, h.CACertPath, h.ClientCertPath, h.ClientKeyPath); err != nil {
				return nil, err
			}
		}

		return transport, nil
	}()

	if err != nil {
		return nil, err
	}

	if h.Username != "" {
		roundTripper = &auth.TransportWithBasicAuth{
			RoundTripper: roundTripper,
			Username:     h.Username,
			Password:     h.Password,
		}
	}

	return &http.Client{
		Timeout:   h.HTTPTimeout,
		Transport: roundTripper,
	}, nil
}
