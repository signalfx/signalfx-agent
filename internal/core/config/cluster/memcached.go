package cluster

import (
	"errors"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/signalfx/signalfx-agent/internal/core/cluster"
)

// MemcachedConfig is the config for the Memcached clustering option.
type MemcachedConfig struct {
	// The Memcached hosts to use.  The list must match on each agent instance
	// in the same "cluster".
	Hosts []string `yaml:"hosts"`
	// How long an agent that claims a key will keep it.  No other agents can
	// obtain it within this amount of time after the agent gets it or renews
	// it.
	Expiration time.Duration `yaml:"expiration" default:"60s"`
	// How often for non-elected agent instances to retry getting the key
	RetryTimeout time.Duration `yaml:"retryTimeout" default:"30s"`
	// How long after obtaining/renewing a claim that the elected agent
	// instance will renew it again.
	RenewalDeadline time.Duration `yaml:"renewalDeadline" default:"45s"`
}

func (kc *MemcachedConfig) NewElector() (cluster.Elector, error) {
	if len(kc.Hosts) == 0 {
		return nil, errors.New("must specify at least one memcached server")
	}
	client := memcache.New(kc.Hosts...)

	elector := cluster.NewMemcachedElector(client)
	elector.Expiration = kc.Expiration
	elector.RetryTimeout = kc.RetryTimeout
	elector.RenewalDeadline = kc.RenewalDeadline
	return elector, nil
}
