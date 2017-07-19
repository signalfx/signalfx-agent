package core

// Do an import of all of the built-in observers and monitors until we get a
// proper plugin system

import (
	_ "github.com/signalfx/neo-agent/monitors/cadvisor"
	_ "github.com/signalfx/neo-agent/monitors/collectd/metadata"
	_ "github.com/signalfx/neo-agent/monitors/collectd/redis"
	_ "github.com/signalfx/neo-agent/monitors/collectd/writehttp"
	_ "github.com/signalfx/neo-agent/monitors/kubernetes"

	_ "github.com/signalfx/neo-agent/observers/docker"
	_ "github.com/signalfx/neo-agent/observers/file"
	_ "github.com/signalfx/neo-agent/observers/kubernetes"
)
