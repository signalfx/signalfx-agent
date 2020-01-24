package httpclient

import (
	"crypto/tls"
	"net/http"

	"github.com/signalfx/signalfx-agent/pkg/core/common/auth"
	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
)

// HTTPConfig can be embedded inside a monitor config.
type HTTPConfig struct {
	// HTTP timeout duration for both read and writes. This should be a
	// duration string that is accepted by https://golang.org/pkg/time/#ParseDuration
	HTTPTimeout timeutil.Duration `yaml:"httpTimeout" default:"10s"`

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
	return h.BuildCustomizeTransport(nil)
}

// Build returns a configured http.Client but applies the provided cb
// function after configuring it to apply any custom configuration
// to the underlying HTTPTransport.
func (h *HTTPConfig) BuildCustomizeTransport(cb func(t *http.Transport)) (*http.Client, error) {
	transport, err := func() (*http.Transport, error) {
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

	// Customize on underlying transport instance before possibly wrapping in auth below.
	if cb != nil {
		cb(transport)
	}

	var roundTripper http.RoundTripper = transport

	if h.Username != "" {
		roundTripper = &auth.TransportWithBasicAuth{
			RoundTripper: roundTripper,
			Username:     h.Username,
			Password:     h.Password,
		}
	}

	return &http.Client{
		Timeout:   h.HTTPTimeout.AsDuration(),
		Transport: roundTripper,
	}, nil
}
