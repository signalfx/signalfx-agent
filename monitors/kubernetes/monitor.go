// This plugin collects high level metrics about a K8s cluster and sends them
// to SignalFx.  The basic technique is to pull data from the K8s API and keep
// up-to-date copies of datapoints for each metric that we collect and then
// ship them off at the end of each reporting interval.  The K8s streaming
// watch API is used to effeciently maintain the state between read intervals
// (see `clusterstate.go`).

// This plugin should only be run at one place in the cluster, or else metrics
// would be duplicated.  This plugin supports two ways of ensuring that:
//
// 1) With the default configuration, this plugin will watch the current list
// of our agent pods, and if and only if it is the first pod in the list,
// sorted alphabetically by pod name ascending, it will be a reporter. Each
// instance of the agent will check upon each reporting interval whether it is
// the first such pod and begin reporting if it finds that it has become the
// reporter.  This method requires one long-running connection to the K8s API
// server per node (assuming the agent is running on all nodes).
//
// 2) You can simply pass a config flag `alwaysClusterReporter` with value of
// `true` to this plugin and it will always report cluster metrics.  This
// method uses less cluster resources (e.g. network sockets, watches on the api
// server) but requires special case configuration for a single agent in the
// cluster, which may be more error prone.
//
// This plugin requires read-only access to the K8s API.
package kubernetes

import (
	"fmt"
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sync"
)

const (
	monitorType = "kubernetes-cluster-metrics"
)

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

type AuthType string

const (
	AuthTypeNone           AuthType = "none"
	AuthTypeTLS            AuthType = "tls"
	AuthTypeServiceAccount AuthType = "serviceAccount"
)

type Config struct {
	config.MonitorConfig
	AlwaysClusterReporter bool
	ClusterName           string `default:"default-cluster"`

	KubernetesAPI struct {
		AuthType       AuthType `default:"serviceAccount"`
		ClientCertPath string
		ClientKeyPath  string
		CACertPath     string `yaml:"caCertPath,omitempty"`
	} `yaml:"kubernetesAPI"`
}

func (c *Config) Validate() bool {
	valid := true
	apiConf := c.KubernetesAPI
	if apiConf.AuthType == AuthTypeTLS && (apiConf.ClientCertPath == "" || apiConf.ClientKeyPath == "") {
		logger.Error("For TLS auth, you must set both the kubernetesAPI.clientCertPath " +
			"and kubernetesAPI.clientKeyPath config values")
		valid = false
	}
	return valid
}

// Monitor makes a distinction between the plugin and the monitor
// itself for less coupling to neo-agent in case we split it out at some point
type Monitor struct {
	config  *Config
	monitor *Kubernetes
	lock    sync.Mutex
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// This can take configuration if needed for other types of auth
func makeK8sClient(config *Config) (*k8s.Clientset, error) {
	apiConf := config.KubernetesAPI
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

	client, err := k8s.NewForConfig(authConf)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Configure is called by the plugin framework when configuration changes
func (m *Monitor) Configure(config *Config) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.monitor != nil {
		m.Shutdown()
	}

	k8sClient, err := makeK8sClient(config)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not create K8s API client")
		return false
	}

	sfxClient := NewSFXClient(map[string]string{
		"metric_source":      "kubernetes",
		"kubernetes_cluster": config.ClusterName,
	})

	sfxClient.AuthToken = config.SignalFxAccessToken

	endpointURL, err := config.IngestURL.Parse("v2/datapoint")
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not construct ingest URL for kubernetes cluster monitor")
		return false
	}
	sfxClient.DatapointEndpoint = endpointURL.String()

	var thisPodName string
	// We need to know the pod name if we aren't always reporting
	if !config.AlwaysClusterReporter {
		var ok bool
		thisPodName, ok = os.LookupEnv("MY_POD_NAME")
		if !ok {
			logger.Error("This pod's name is not known! Please inject the envvar MY_POD_NAME " +
				"via a config fieldRef in your K8s agent resource config")
			return false
		}
	}

	m.monitor = NewKubernetes(
		k8sClient,
		sfxClient,
		uint(config.IntervalSeconds),
		config.AlwaysClusterReporter,
		thisPodName)

	m.monitor.Filter = config.Filter

	m.monitor.Start()

	return true
}

// Shutdown halts everything that is syncing
func (m *Monitor) Shutdown() {
	if m.monitor != nil {
		m.monitor.Stop()
	}
}
