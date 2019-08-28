package cluster

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/internal/core/cluster"
	"github.com/signalfx/signalfx-agent/internal/core/common/kubernetes"
)

// KubernetesConfig is some common K8s config used by multiple monitors
type KubernetesConfig struct {
	// Config for the K8s API client
	KubernetesAPI *kubernetes.APIConfig `yaml:"kubernetesAPI" default:"{}"`
}

func (kc *KubernetesConfig) NewElector() (cluster.Elector, error) {
	clientSet, err := kubernetes.MakeClient(kc.KubernetesAPI)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes client for leader election: %s", err)
	}

	return cluster.NewKubernetesElector(clientSet.CoreV1()), nil
}
