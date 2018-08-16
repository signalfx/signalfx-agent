package core

// Do an import of all of the built-in observers and monitors
// that apply to all platforms until we get a proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/signalfx-agent/internal/monitors/cadvisor"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/conviva"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/docker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/internalmetrics"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/metadata/hostmetadata"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/processlist"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/traceforwarder"
	_ "github.com/signalfx/signalfx-agent/internal/observers/docker"
	_ "github.com/signalfx/signalfx-agent/internal/observers/file"
	_ "github.com/signalfx/signalfx-agent/internal/observers/host"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubelet"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubernetes"
)
