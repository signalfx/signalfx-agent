package cluster

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/signalfx/signalfx-agent/internal/core/cluster"
)

// ClusteringConfig describes how to cluster the agents, if desired.
type ClusteringConfig struct {
	// How to cluster the agents.
	Type       string            `yaml:"type" validate:"oneof=kubernetes memcached"`
	Kubernetes *KubernetesConfig `yaml:"kubernetes"`
	Memcached  *MemcachedConfig  `yaml:"memcached"`
}

func (cc *ClusteringConfig) CreateElector(ctx context.Context, hostIDDims map[string]string) (*cluster.MultiElector, error) {
	var agentID string
	for k, v := range hostIDDims {
		if v == "" {
			continue
		}
		agentID = fmt.Sprintf("%s_%s_%d", k, v, os.Getpid())
	}
	if agentID == "" {
		return nil, errors.New("unique agent instance id could not be determined")
	}

	var elector cluster.Elector
	var err error
	switch cc.Type {
	case "kubernetes":
		if cc.Kubernetes == nil {
			cc.Kubernetes = &KubernetesConfig{}
		}
		elector, err = cc.Kubernetes.NewElector()
	case "memcached":
		if cc.Memcached == nil {
			return nil, errors.New("memcached clustering must be configured")
		}
		elector, err = cc.Memcached.NewElector()
	default:
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not create elector implementation: %v", err)
	}

	return cluster.NewMultiElector(ctx, agentID, elector), nil
}
