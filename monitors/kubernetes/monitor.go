// Package kubernetes contains a Kubernetes monitor.
//
// This plugin collects high level metrics about a K8s cluster and sends them
// to SignalFx.  The basic technique is to pull data from the K8s API and keep
// up-to-date copies of datapoints for each metric that we collect and then
// ship them off at the end of each reporting interval.  The K8s streaming
// watch API is used to effeciently maintain the state between read intervals
// (see `clusterstate.go`).
//
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
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/common/kubernetes"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"

	"sync"
)

const (
	monitorType = "kubernetes-cluster"
)

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for the K8s monitor
type Config struct {
	config.MonitorConfig
	AlwaysClusterReporter bool                            `yaml:"alwaysClusterReporter"`
	KubernetesAPI         *kubernetes.KubernetesAPIConfig `yaml:"kubernetesAPI" default:"{}"`
}

// Validate the k8s-specific config
func (c *Config) Validate() bool {
	if !c.KubernetesAPI.Validate() {
		return false
	}
	return true
}

// Monitor makes a distinction between the plugin and the monitor
// itself for less coupling to neo-agent in case we split it out at some point
type Monitor struct {
	config  *Config
	monitor *Kubernetes
	lock    sync.Mutex
	DPs     chan<- *datapoint.Datapoint
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure is called by the plugin framework when configuration changes
func (m *Monitor) Configure(config *Config) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.monitor != nil {
		m.Shutdown()
	}

	k8sClient, err := kubernetes.MakeClient(config.KubernetesAPI)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not create K8s API client")
		return false
	}

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
		m.DPs,
		uint(config.IntervalSeconds),
		config.AlwaysClusterReporter,
		thisPodName)

	m.monitor.Start()

	return true
}

// Shutdown halts everything that is syncing
func (m *Monitor) Shutdown() {
	if m.monitor != nil {
		m.monitor.Stop()
	}
}
