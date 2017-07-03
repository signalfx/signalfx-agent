package kubernetes

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/signalfx/golib/sfxclient"
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
// 2) You can simply pass a config flag `isClusterReporter` with value of
// `true` to this plugin and it will always report cluster metrics.  This
// method uses less cluster resources (e.g. network sockets, watches on the api
// server) but requires special case configuration for a single agent in the
// cluster, which may be more error prone.
//
// This plugin requires read-only access to the K8s API.

// Plugin makes a distinction between the plugin and the monitor
// itself for less coupling to neo-agent in case we split it out at some point
type Plugin struct {
	plugins.Plugin
	monitor *Kubernetes
}

const (
	pluginType = "monitors/kubernetes"
)

func init() {
	plugins.Register(pluginType, NewPlugin)
}

// NewPlugin makes a new instance of the plugin
func NewPlugin(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}

	kmp := &Plugin{
		Plugin:  plugin,
		monitor: nil, // This gets set in Configure
	}

	err = kmp.Configure(config)
	if err != nil {
		return nil, err
	}

	return kmp, nil
}

// This can take configuration if needed for other types of auth
func makeK8sClient(config *viper.Viper) (*k8s.Clientset, error) {
	authType := config.GetString("authType")
	noVerify := config.GetBool("tls.skipVerify")

	var authConf *rest.Config
	var err error

	switch authType {
	// Mainly for testing purposes
	case "none":
		host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
		if len(host) == 0 || len(port) == 0 {
			return nil, fmt.Errorf("unable to load k8s config, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined")
		}
		authConf = &rest.Config{
			Host: "https://" + net.JoinHostPort(host, port),
		}
	default:
		log.Print("No k8s API auth specified, defaulting to service accounts")
		// This should work for most clusters but other auth types can be added
		authConf, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	authConf.Insecure = noVerify

	client, err := k8s.NewForConfig(authConf)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Configure is called by the plugin framework when configuration changes
func (kmp *Plugin) Configure(config *viper.Viper) error {
	kmp.Stop()

	kmp.Config = config
	kmp.Config.SetDefault("alwaysReport", false)
	kmp.Config.SetDefault("intervalSeconds", 10)

	k8sClient, err := makeK8sClient(config)
	if err != nil {
		return err
	}

	interval := uint(config.GetInt("intervalSeconds"))

	sfxClient := sfxclient.NewHTTPSink()
	sfxAccessToken, err := secrets.EnvSecret("SFX_ACCESS_TOKEN")
	if err != nil {
		return err
	}
	sfxClient.AuthToken = sfxAccessToken

	sfxIngestURL := config.GetString("ingesturl")
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

	alwaysReport := config.GetBool("alwaysReport")

	var thisPodName string
	// We need to know the pod name if we aren't always reporting
	if !alwaysReport {
		var ok bool
		thisPodName, ok = os.LookupEnv("MY_POD_NAME")
		if !ok {
			return fmt.Errorf("This pod's name not set! Please inject the envvar MY_POD_NAME " +
				"via a config fieldRef")
		}
	}

	kmp.monitor = NewKubernetes(k8sClient, sfxClient, interval, alwaysReport, thisPodName)

	kmp.monitor.MetricFilter = newFilterSet(config.GetStringSlice("metricFilter"))
	kmp.monitor.NamespaceFilter = newFilterSet(config.GetStringSlice("namespaceFilter"))

	return nil
}

// Stop halts everything that is syncing
func (kmp *Plugin) Stop() {
	if kmp.monitor != nil {
		kmp.monitor.Stop()
	}
}

// Start begins the data collection
func (kmp *Plugin) Start() error {
	kmp.monitor.Start()
	return nil
}
