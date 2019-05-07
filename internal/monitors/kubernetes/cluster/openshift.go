package cluster

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/cluster/meta"
)

func init() {
	monitors.Register(&meta.OpenshiftClusterMonitorMetadata,
		func() interface{} { return &Monitor{distribution: OpenShift} }, &Config{})
}
