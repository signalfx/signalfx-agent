package kubelet

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
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
	// URL of the Kubelet instance.  This will default to `https://<current
	// node hostname>:10250` if not provided.
	URL string `yaml:"url"`
	// Can be `none` for no auth, `tls` for TLS client cert auth, or
	// `serviceAccount` to use the pod's default service account token to
	// authenticate.
	AuthType AuthType `yaml:"authType" default:"none"`
	// Whether to skip verification of the Kubelet's TLS cert
	SkipVerify *bool `yaml:"skipVerify" default:"true"`
	// Path to the CA cert that has signed the Kubelet's TLS cert, unnecessary
	// if `skipVerify` is set to false.
	CACertPath string `yaml:"caCertPath"`
	// Path to the client TLS cert to use if `authType` is set to `tls`
	ClientCertPath string `yaml:"clientCertPath"`
	// Path to the client TLS key to use if `authType` is set to `tls`
	ClientKeyPath string `yaml:"clientKeyPath"`
	// Whether to log the raw cadvisor response at the debug level for
	// debugging purposes.
	LogResponses bool `yaml:"logResponses" default:"false"`
}

// Client is a wrapper around http.Client that injects the right auth to every
// request.
type Client struct {
	*http.Client
	config *APIConfig
}

// NewClient creates a new client with the given config
func NewClient(kubeletAPI *APIConfig) (*Client, error) {
	var certs *x509.CertPool
	if runtime.GOOS != "windows" {
		var err error
		certs, err = x509.SystemCertPool()
		if err != nil {
			return nil, errors.WithMessage(err, "Could not load system x509 cert pool")
		}
	}
	if kubeletAPI.URL == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		kubeletAPI.URL = fmt.Sprintf("https://%s:10250", hostname)
	}

	tlsConfig := &tls.Config{}
	if kubeletAPI.SkipVerify != nil {
		tlsConfig.InsecureSkipVerify = *kubeletAPI.SkipVerify
	}

	var transport http.RoundTripper = &(*http.DefaultTransport.(*http.Transport))
	if kubeletAPI.AuthType == AuthTypeTLS {
		if kubeletAPI.CACertPath != "" && certs != nil {
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
				return nil, errors.WithMessage(err,
					fmt.Sprintf("Kubelet client cert/key could not be loaded from %s/%s",
						clientKeyPath, clientCertPath))
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
			return nil, errors.WithMessage(err, fmt.Sprintf("Could not load root CA config from %s", rootCAFile))
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

// DoRequestAndSetValue does a request to the Kubelet and populates the `value`
// by deserializing the JSON in the response.
func (kc *Client) DoRequestAndSetValue(req *http.Request, value interface{}) error {
	response, err := kc.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read Kubelet response body - %v", err)
	}

	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Kubelet request resulted in 404: %s", req.URL.String())
	} else if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Kubelet request failed - %q, response: %q", response.Status, string(body))
	}

	if kc.config.LogResponses {
		log.Debugf("Raw response from Kubelet url %s: %s", req.URL.String(), string(body))
	}

	err = json.Unmarshal(body, value)
	if err != nil {
		return fmt.Errorf("Failed to parse Kubelet output. Response: %q. Error: %v", string(body), err)
	}
	return nil
}
