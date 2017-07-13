package kubernetes

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sync"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/secrets"
	"github.com/spf13/viper"
)

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

// Plugin makes a distinction between the plugin and the monitor
// itself for less coupling to neo-agent in case we split it out at some point
type Plugin struct {
	monitor *Kubernetes
	lock    sync.Mutex
}

const (
	pluginType = "monitors/kubernetes"
)

func init() {
	plugins.Register(pluginType, func() interface{} { return &Plugin{} })
}

// This can take configuration if needed for other types of auth
func makeK8sClient(config *viper.Viper) (*k8s.Clientset, error) {
	authType := config.GetString("authType")

	var authConf *rest.Config
	var err error

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("unable to load k8s config, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
	}
	k8sHost := "https://" + net.JoinHostPort(host, port)

	switch authType {
	// Mainly for testing purposes
	case "none":
		authConf = &rest.Config{
			Host: k8sHost,
		}
		authConf.Insecure = true
	case "tls":
		if !config.IsSet("tls.clientCert") || !config.IsSet("tls.clientKey") {
			return nil, fmt.Errorf("For TLS auth, you must set both the "+
			"kubernetesAPI.tls.clientCert and kubernetesAPI.tls.clientKey "+
			"config values")
		}
		authConf = &rest.Config{
			Host: k8sHost,
			TLSClientConfig: rest.TLSClientConfig{
				CertFile: config.GetString("tls.clientCert"),
				KeyFile: config.GetString("tls.clientKey"),
				CAFile: config.GetString("tls.caCert"),
			},
		}
	default:
		log.Print("No k8s API auth specified, defaulting to service accounts")
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
func (kmp *Plugin) Configure(config *viper.Viper) error {
	kmp.lock.Lock()
	defer kmp.lock.Unlock()
	if kmp.monitor != nil {
		// lock on configure
		kmp.Shutdown()
	}

	config.SetDefault("alwaysClusterReporter", false)
	config.SetDefault("intervalSeconds", 10)

	k8sClient, err := makeK8sClient(config)
	if err != nil {
		return err
	}

	interval := uint(config.GetInt("intervalSeconds"))

	sfxClient := NewSFXClient(map[string]string{
		"metric_source": "kubernetes",
		"kubernetes_cluster": config.GetString("clusterName"),
	})
	sfxAccessToken, err := secrets.EnvSecret("SFX_ACCESS_TOKEN")
	if err != nil {
		return err
	}
	sfxClient.AuthToken = sfxAccessToken

	// TODO: make ingesturl accessible in a better way, right now it's a global viper variable
	sfxIngestURL := viper.GetString("ingesturl")
	if sfxIngestURL != "" {
		baseURL, err := url.Parse(sfxIngestURL)
		if err != nil {
			return fmt.Errorf("Could not parse SignalFx ingest url: %s", err)
		}

		endpointURL, err := baseURL.Parse("v2/datapoint")
		if err != nil {
			return fmt.Errorf("Something went horribly wrong: %s", err)
		}
		sfxClient.DatapointEndpoint = endpointURL.String()
	}

	alwaysClusterReporter := config.GetBool("alwaysClusterReporter")

	var thisPodName string
	// We need to know the pod name if we aren't always reporting
	if !alwaysClusterReporter {
		var ok bool
		thisPodName, ok = os.LookupEnv("MY_POD_NAME")
		if !ok {
			return fmt.Errorf("This pod's name not set! Please inject the envvar MY_POD_NAME " +
				"via a config fieldRef")
		}
	}

	kmp.monitor = NewKubernetes(k8sClient, sfxClient, interval, alwaysClusterReporter, thisPodName)

	kmp.monitor.MetricFilter = newFilterSet(config.GetStringSlice("metricFilter"))
	kmp.monitor.NamespaceFilter = newFilterSet(config.GetStringSlice("namespaceFilter"))
	log.Printf("K8s Cluster Metric Filters: %#v", config.GetStringSlice("metricFilter"))
	log.Printf("K8s Cluster Namespace Filters: %#v", config.GetStringSlice("namespaceFilter"))

	kmp.monitor.Start()

	return nil
}

// Shutdown halts everything that is syncing
func (kmp *Plugin) Shutdown() {
	if kmp.monitor != nil {
		kmp.monitor.Stop()
	}
}
