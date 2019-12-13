package cluster

import (
	"github.com/signalfx/signalfx-agent/internal/monitors/kubernetes/cluster/meta"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

func init() {
	monitors.Register(&meta.OpenshiftClusterMonitorMetadata,
		func() interface{} { return &Monitor{distribution: OpenShift} }, &Config{})
}
