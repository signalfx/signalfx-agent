package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"bytes"

	"sync"

	"github.com/docker/libkv/store"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultInterval is used if not configured
	DefaultInterval = 10
	// DefaultPipeline is used if not configured
	DefaultPipeline = "file"
	// DefaultPollingInterval is the interval in seconds between checking configuration files for changes
	DefaultPollingInterval = 5
	// EnvPrefix is the environment variable prefix
	EnvPrefix = "SFX"

	envMergeConfig = "SFX_MERGE_CONFIG"
	envUserConfig  = "SFX_USER_CONFIG"
)

const (
	// WatchInitial is the initial watch state
	WatchInitial = iota
	// WatchFailed is the watch failed state
	WatchFailed
	// Watching is the normal watching state
	Watching
)

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer   = strings.NewReplacer(".", "_", "-", "_")
	configTimeout = 10 * time.Second
)

// Label -
type Label struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value,omitempty"`
}

type userConfig struct {
	Filter *struct {
		DockerContainerNames     []string `yaml:"dockerContainerNames,omitempty"`
		Images                   []string `yaml:"images,omitempty,omitempty"`
		KubernetesContainerNames []string `yaml:"kubernetesContainerNames,omitempty"`
		KubernetesPodNames       []string `yaml:"kubernetesPodNames,omitempty"`
		KubernetesNamespaces     []string `yaml:"kubernetesNamespaces,omitemtpy"`
		Labels                   []*Label `yaml:"labels,omitempty"`
	} `yaml:"filterContianerMetrics,omitempty"`
	Proxy *struct {
		HTTP  string
		HTTPS string
		Skip  string
	}
	Kubernetes *struct {
		TLS struct {
			SkipVerify bool   `yaml:"skipVerify"`
			ClientCert string `yaml:"clientCert"`
			ClientKey  string `yaml:"clientKey"`
			CACert     string `yaml:"caCert"`
		} `yaml:"tls"`
		Role        string
		Cluster     string
		CAdvisorURL string `yaml:"cadvisorURL,omitempty"`
	}
	Mesosphere *struct {
		Cluster      string
		Role         string
		SystemHealth bool `yaml:"systemHealth,omitempty"`
		Verbose      bool `yaml:"verbose,omitempty"`
	}
}

// getMergeConfigs returns list of config files to merge from the environment
// variable
func getMergeConfigs() []string {
	var configs []string

	if merge := os.Getenv(envMergeConfig); len(merge) > 1 {
		for _, mergeFile := range strings.Split(merge, ",") {
			configs = append(configs, mergeFile)
		}
	}

	return configs
}

func loadUserConfig(pair *store.KVPair) error {
	var usercon userConfig
	if err := yaml.Unmarshal(pair.Value, &usercon); err != nil {
		return err
	}
	// create cadvisor configuration map
	cadvisor := map[string]interface{}{}

	// create docker-default configuration map
	dockerDefaults := map[string]interface{}{}

	// create staticplugins configuration map
	staticPlugins := map[string]interface{}{}

	// create collectd configuration map
	collectd := map[string]interface{}{
		"staticPlugins": staticPlugins,
	}

	// create plugins configuration map
	plugins := map[string]interface{}{
		"collectd": collectd,
	}

	// create dims plugin configuration map
	dims := map[string]string{}

	// create viper configuration map
	v := map[string]interface{}{
		"plugins":    plugins,
		"dimensions": dims,
	}

	if usercon.Kubernetes != nil && usercon.Mesosphere != nil {
		return errors.New("mesosphere and kubernetes cannot both be set")
	}

	// configure filters
	if filters := usercon.Filter; filters != nil {
		// assign image filters
		if len(filters.Images) != 0 {
			dockerDefaults["excludedImages"] = filters.Images
			cadvisor["excludedImages"] = filters.Images
		}
		// assign docker container name filter
		if len(filters.DockerContainerNames) != 0 {
			dockerDefaults["excludedNames"] = filters.DockerContainerNames
			cadvisor["excludedNames"] = filters.DockerContainerNames
		}
		// configure the label filters
		if len(filters.Labels) != 0 || len(filters.KubernetesNamespaces) != 0 || len(filters.KubernetesContainerNames) != 0 || len(filters.KubernetesPodNames) != 0 {
			// assign namespaces to labels because k8s namespace is actually a label
			for _, namespace := range filters.KubernetesNamespaces {
				filters.Labels = append(filters.Labels, &Label{Key: "^io.kubernetes.pod.namespace$", Value: namespace})
			}
			// assign k8s container name to labels because k8s container name is actually a label
			for _, containerName := range filters.KubernetesContainerNames {
				filters.Labels = append(filters.Labels, &Label{Key: "^io.kubernetes.container.name$", Value: containerName})
			}

			// assign k8s pod name to labels because k8s podname is actually a label
			for _, podName := range filters.KubernetesPodNames {
				filters.Labels = append(filters.Labels, &Label{Key: "^io.kubernetes.pod.name$", Value: podName})
			}

			// if there are labels add them
			if len(filters.Labels) != 0 {
				// append the lables filter
				dockerDefaults["excludedLabels"] = filters.Labels
				cadvisor["excludedLabels"] = filters.Labels
			}
		}
		// Since the filters are set let's set docker-default
		staticPlugins["docker-default"] = dockerDefaults
	}

	if kube := usercon.Kubernetes; kube != nil {
		if kube.Cluster == "" {
			return errors.New("kubernetes.cluster missing")
		}
		if kube.Role != "worker" && kube.Role != "master" {
			return errors.New("kubernetes.role must be worker or master")
		}

		dims["kubernetes_cluster"] = kube.Cluster
		dims["kubernetes_role"] = kube.Role

		if kube.Role == "worker" {
			if kube.CAdvisorURL != "" {
				// add the config from user config to cadvisor plugin config
				cadvisor["cadvisorurl"] = kube.CAdvisorURL
				// add config to plugins config
				plugins["cadvisor"] = cadvisor
			}
			kubernetes := map[string]interface{}{}

			tls := kube.TLS
			tlsConfig := map[string]interface{}{
				"caCert":     tls.CACert,
				"skipVerify": tls.SkipVerify,
				"clientCert": tls.ClientCert,
				"clientKey":  tls.ClientKey,
			}
			kubernetes["tls"] = tlsConfig
			plugins["kubernetes"] = kubernetes
		}
	}

	if proxy := usercon.Proxy; proxy != nil {
		proxyConfig := map[string]string{}
		proxyConfig["http"] = proxy.HTTP
		proxyConfig["https"] = proxy.HTTPS
		proxyConfig["skip"] = proxy.Skip
		plugins["proxy"] = proxyConfig
	}

	if mesos := usercon.Mesosphere; mesos != nil {
		var mesosPort int
		var mesosIDDimName string
		var mesosID string

		client := NewMesosClient(viper.GetViper())
		if mesos.Role == "master" {
			mesosPort = 5050
			mesosIDDimName = "mesos_master"
		} else if mesos.Role == "worker" {
			mesosIDDimName = "mesos_agent"
			mesosPort = 5051
		} else {
			return errors.New("mesosphere role must be specified")
		}

		if err := client.Configure(viper.GetViper(), mesosPort); err != nil {
			return fmt.Errorf("unable to configure mesos client at configuration time: %s", err)
		}

		ID, _ := client.GetID()
		mesosID = ID.ID

		if mesos.Cluster == "" {
			return errors.New("mesosphere.cluster must be set")
		}
		dims["mesos_cluster"] = mesos.Cluster
		dims["mesos_role"] = mesos.Role
		dims[mesosIDDimName] = mesosID

		// Set the cluster name for the mesos default plugin config
		staticPlugins["mesos"] = map[string]interface{}{
			"cluster":      mesos.Cluster,
			"systemhealth": mesos.SystemHealth,
			"verbose":      mesos.Verbose,
		}
	}

	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)

	if err := viper.MergeConfig(r); err != nil {
		return err
	}

	return nil
}

// load takes a config file in path and performs a reload when the path has changed
func load(path string, reload chan<- struct{}, cb func(pair *store.KVPair) error, mutex *sync.Mutex) error {
	source, path, err := Stores.Get(path)
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		if i != 0 {
			log.Printf("sleeping 5 seconds before retrying EnsureExist")
			time.Sleep(5 * time.Second)
		}
		if err := EnsureExists(source, path); err != nil {
			log.Printf("error for EnsureExist: %s", err)
		} else {
			goto Success
		}
	}

	return fmt.Errorf("failed ensuring %s exists", path)

Success:
	ch, err := ReconnectWatch(source, path, nil)

	if err != nil {
		return err
	}

	select {
	case pair := <-ch:
		log.Printf("config %s loaded", path)
		if err := cb(pair); err != nil {
			return err
		}
	case <-time.After(configTimeout):
		return fmt.Errorf("failed getting initial configuration for %s", path)
	}

	go func() {
		for pair := range ch {
			mutex.Lock()
			if err := cb(pair); err != nil {
				log.Printf("error in callback for %s: %s", path, err)
			}
			mutex.Unlock()
			reload <- struct{}{}
		}
		log.Printf("error: watch stopped for %s", path)
	}()

	return nil

}

// Init loads the config from configfile as well as any merge files from
// environment variable
func Init(configfile string, reload chan<- struct{}, mutex *sync.Mutex) error {
	// Lock so that goroutines kicked off don't modify viper while we're still
	// synchronously loading.
	mutex.Lock()
	defer mutex.Unlock()

	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("pipeline", DefaultPipeline)
	viper.SetDefault("pollingInterval", DefaultPollingInterval)
	viper.SetDefault("ingesturl", "https://ingest.signalfx.com")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(EnvReplacer)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetConfigFile(configfile)

	err := load(configfile, reload, func(pair *store.KVPair) error {
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("error reading agent config: %s", err)
			return err
		}

		return nil
	}, mutex)

	if err != nil {
		return err
	}

	// Configure stores after the base config file has been loaded.
	if err := Stores.Config(viper.Sub("stores")); err != nil {
		return err
	}

	for _, mergeFile := range getMergeConfigs() {
		log.Printf("loading merged config from %s", mergeFile)

		err := load(mergeFile, reload, func(pair *store.KVPair) error {
			log.Printf("%s changed", pair.Key)

			reader := bytes.NewReader(pair.Value)
			if err := viper.MergeConfig(reader); err != nil {
				log.Printf("error merging changes to %s", pair.Key)
				return err
			}

			return nil
		}, mutex)

		if err != nil {
			return err
		}
	}

	// Load user config.
	if userConfigFile := os.Getenv(envUserConfig); userConfigFile != "" {
		log.Printf("loading user configuration from %s", userConfigFile)

		err := load(userConfigFile, reload, func(pair *store.KVPair) error {
			log.Printf("%s changed", userConfigFile)

			if err := loadUserConfig(pair); err != nil {
				log.Printf("failed loading user configuration for %s", userConfigFile)
				return err
			}

			return nil
		}, mutex)

		if err != nil {
			return err
		}
	}

	return nil
}
