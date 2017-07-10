package userconfig

import (
	"errors"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

const k8sWorker = "worker"

// Kubernetes user configuration struct for kubernetes
type Kubernetes struct {
	Role                 string
	Cluster              string
	CAdvisorURL          string          `yaml:"cadvisorURL,omitempty"`
	CAdvisorMetricFilter []string        `yaml:"cadvisorDisabledMetrics,omitempty"`
	CAdvisorDataSendRate int             `yaml:"cadvisorSendRate,omitempty"`
	ClusterMetrics       *ClusterMetrics `yaml:"clusterMetrics,omitempty"`
	KubeletAPI           *struct {
		TLS *TLS `yaml:"tls,omitempty"`
	} `yaml:"kubeletAPI,omitempty"`
	KubernetesAPI *struct {
		AuthType string   `yaml:"authType,omitempty"`
		TLS      *TLS     `yaml:"tls,omitempty"`
	} `yaml:"kubernetesAPI,omitempty"`
}

// LoadYAML loads a yaml file
func (k *Kubernetes) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &k); err != nil {
			return err
		}
	}
	return err
}

// IsValid checks whether the kubernetes cluster is set and has a valid role
func (k *Kubernetes) IsValid() (bool, error) {
	if k.Cluster == "" {
		return false, errors.New("kubernetes.cluster missing")
	}
	if k.Role != k8sWorker && k.Role != "master" {
		return false, errors.New("kubernetes.role must be worker or master")
	}
	return true, nil
}

// Parse parses the configuration into a supplied map
func (k *Kubernetes) Parse(kubernetes map[string]interface{}) error {
	if ok, err := k.IsValid(); !ok {
		return err
	}
	if k.Role == k8sWorker {
		if k.KubeletAPI != nil {
			var tls = map[string]interface{}{}
			if k.KubeletAPI.TLS != nil {
				k.KubeletAPI.TLS.Parse(tls)
			}
			if len(tls) > 0 {
				kubernetes["tls"] = tls
			}
		}
	}

	return nil
}

// ParseDimensions parses dimensions into the supplied map
func (k *Kubernetes) ParseDimensions(dims map[string]string) error {
	if ok, err := k.IsValid(); !ok {
		return err
	}

	dims["kubernetes_cluster"] = k.Cluster
	dims["kubernetes_role"] = k.Role

	return nil
}

// ParseClusterMetrics parses configurations for the cluster metrics collector
func (k *Kubernetes) ParseClusterMetrics(clusterMetrics map[string]interface{}) error {
	if ok, err := k.IsValid(); !ok {
		return err
	}
	if k.ClusterMetrics != nil {
		k.ClusterMetrics.Parse(clusterMetrics)
	}
	if k.Cluster != "" {
		clusterMetrics["clusterName"] = k.Cluster
	}
	if k.KubernetesAPI != nil {
		if k.KubernetesAPI.AuthType != "" {
			clusterMetrics["authType"] = k.KubernetesAPI.AuthType
		}

		var tls = map[string]interface{}{}
		if k.KubernetesAPI.TLS != nil {
			k.KubernetesAPI.TLS.Parse(tls)
			if len(tls) > 0 {
				clusterMetrics["tls"] = tls
			}
		}
	}
	return nil
}

// ParseCAdvisor parses cadvisor configurations into the supplied map
func (k *Kubernetes) ParseCAdvisor(cadvisor map[string]interface{}) error {
	if ok, err := k.IsValid(); !ok {
		return err
	}
	if k.Role == k8sWorker {
		if k.CAdvisorURL != "" || len(k.CAdvisorMetricFilter) > 0 || k.CAdvisorDataSendRate != 0 {
			// parse metric names for cadvisor to not collect
			if len(k.CAdvisorMetricFilter) > 0 {
				var filters = map[string]bool{}
				for _, metric := range k.CAdvisorMetricFilter {
					filters[metric] = true
				}
				cadvisor["excludedMetrics"] = filters
			}
			if k.CAdvisorURL != "" {
				// add the config from user config to cadvisor plugin config
				cadvisor["cadvisorurl"] = k.CAdvisorURL
			}
			// set the data send rate for cadvisor
			if k.CAdvisorDataSendRate != 0 {
				cadvisor["dataSendRate"] = k.CAdvisorDataSendRate
			}
		}
	}
	return nil
}
