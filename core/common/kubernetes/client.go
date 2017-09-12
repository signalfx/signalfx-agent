package kubernetes

import (
	"fmt"
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// AuthType describes the type of authentication to use for the K8s API
type AuthType string

const (
	// AuthTypeNone means no auth is required
	AuthTypeNone AuthType = "none"
	// AuthTypeTLS means to use client TLS certs
	AuthTypeTLS AuthType = "tls"
	// AuthTypeServiceAccount means to use the built-in service account that
	// K8s automatically provisions for each pod.
	AuthTypeServiceAccount AuthType = "serviceAccount"
)

// APIConfig contains options relevant to connecting to the K8s API
type APIConfig struct {
	AuthType       AuthType `yaml:"authType" default:"serviceAccount"`
	SkipVerify     bool     `yaml:"skipVerify" default:"false"`
	ClientCertPath string   `yaml:"clientCertPath"`
	ClientKeyPath  string   `yaml:"clientKeyPath"`
	CACertPath     string   `yaml:"caCertPath"`
}

// Validate validates the K8s API config
func (c *APIConfig) Validate() bool {
	if c.AuthType == AuthTypeTLS && (c.ClientCertPath == "" || c.ClientKeyPath == "") {
		log.Error("For TLS auth, you must set both the kubernetesAPI.clientCertPath " +
			"and kubernetesAPI.clientKeyPath config values")
		return false
	}
	return true
}

// MakeClient can take configuration if needed for other types of auth
func MakeClient(apiConf *APIConfig) (*k8s.Clientset, error) {
	authType := apiConf.AuthType

	var authConf *rest.Config
	var err error

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("unable to load k8s config, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	}
	k8sHost := "https://" + net.JoinHostPort(host, port)

	switch authType {
	// Mainly for testing purposes
	case AuthTypeNone:
		authConf = &rest.Config{
			Host: k8sHost,
		}
		authConf.Insecure = true
	case AuthTypeTLS:
		authConf = &rest.Config{
			Host: k8sHost,
			TLSClientConfig: rest.TLSClientConfig{
				CertFile: apiConf.ClientCertPath,
				KeyFile:  apiConf.ClientKeyPath,
				CAFile:   apiConf.CACertPath,
			},
		}
	case AuthTypeServiceAccount:
		// This should work for most clusters but other auth types can be added
		authConf, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	authConf.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		// Don't use system proxy settings since the API is local to the
		// cluster
		if t, ok := rt.(*http.Transport); ok {
			t.Proxy = nil
		}
		return rt
	}

	authConf.Insecure = apiConf.SkipVerify

	client, err := k8s.NewForConfig(authConf)
	if err != nil {
		return nil, err
	}

	return client, nil
}
