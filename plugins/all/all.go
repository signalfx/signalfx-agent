package all

import (
	// Include all plugin packages so that init is called for registration.
	_ "github.com/signalfx/neo-agent/plugins/filters/debug"
	_ "github.com/signalfx/neo-agent/plugins/filters/integrations"
	_ "github.com/signalfx/neo-agent/plugins/filters/proxy"
	_ "github.com/signalfx/neo-agent/plugins/monitors/cadvisor"
	_ "github.com/signalfx/neo-agent/plugins/monitors/collectd"
	_ "github.com/signalfx/neo-agent/plugins/observers/docker"
	_ "github.com/signalfx/neo-agent/plugins/observers/file"
	_ "github.com/signalfx/neo-agent/plugins/observers/kubernetes"
	_ "github.com/signalfx/neo-agent/plugins/observers/mesosphere"
)
