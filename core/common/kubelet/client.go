package kubelet

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// AuthType to use when connecting to kubelet
type AuthType string

const (
	// AuthTypeNone means there is no authentication to kubelet
	AuthTypeNone AuthType = "none"
	// AuthTypeTLS indicates that client TLS auth is desired
	AuthTypeTLS AuthType = "tls"
	// AuthTypeServiceAccount indicates that the default service account token should be used
	AuthTypeServiceAccount AuthType = "serviceAccount"
)

// APIConfig contains config specific to the KubeletAPI
type APIConfig struct {
	URL            string   `yaml:"url"`
	AuthType       AuthType `yaml:"authType" default:"none"`
	SkipVerify     bool     `yaml:"skipVerify" default:"false"`
	CACertPath     string   `yaml:"caCertPath"`
	ClientCertPath string   `yaml:"clientCertPath"`
	ClientKeyPath  string   `yaml:"clientKeyPath"`
}

// Client is a wrapper around http.Client that injects the right auth to every
// request.
type Client struct {
	*http.Client
	config *APIConfig
}

// NewClient creates a new client with the given config
func NewClient(kubeletAPI *APIConfig) (*Client, error) {
	certs, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not load system x509 cert pool")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: kubeletAPI.SkipVerify,
	}

	var transport http.RoundTripper = &(*http.DefaultTransport.(*http.Transport))
	if kubeletAPI.AuthType == AuthTypeTLS {
		if kubeletAPI.CACertPath != "" {
			if err := augmentCertPoolFromCAFile(certs, kubeletAPI.CACertPath); err != nil {
				return nil, err
			}
		}

		var clientCerts []tls.Certificate

		clientCertPath := kubeletAPI.ClientCertPath
		clientKeyPath := kubeletAPI.ClientKeyPath
		if clientCertPath != "" && clientKeyPath != "" {
			cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
			if err != nil {
				return nil, errors.Wrapf(err, "Kubelet client cert/key could not be loaded from %s/%s",
					clientKeyPath, clientCertPath)
			}
			clientCerts = append(clientCerts, cert)
			log.Infof("Configured TLS client cert in %s with key %s", clientCertPath, clientKeyPath)
		}

		tlsConfig.Certificates = clientCerts
		tlsConfig.RootCAs = certs
		tlsConfig.BuildNameToCertificate()
		transport.(*http.Transport).TLSClientConfig = tlsConfig
	} else if kubeletAPI.AuthType == AuthTypeServiceAccount {

		token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
		if err != nil {
			return nil, errors.Wrap(err, "Could not read service account token at default location, are "+
				"you sure service account tokens are mounted into your containers by default?")
		}

		rootCAFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
		if err := augmentCertPoolFromCAFile(certs, rootCAFile); err != nil {
			return nil, errors.Wrapf(err, "Could not load root CA config from %s", rootCAFile)
		}

		tlsConfig.RootCAs = certs
		t := transport.(*http.Transport)
		t.TLSClientConfig = tlsConfig

		transport = &transportWithToken{
			Transport: t,
			token:     string(token),
		}

		log.Debug("Using service account authentication for Kubelet")
	} else {
		transport.(*http.Transport).TLSClientConfig = tlsConfig
	}

	return &Client{
		Client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		config: kubeletAPI,
	}, nil
}

// NewRequest is used to provide a base URL to which paths can be appended.
// Other than the second argument it is identical to the http.NewRequest
// method.
func (kc *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	baseURL := kc.config.URL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(path, "/") {
		baseURL += "/"
	}

	return http.NewRequest(method, baseURL+path, body)
}
