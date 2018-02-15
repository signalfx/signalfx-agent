package meta

import "github.com/signalfx/signalfx-agent/internal/core/config"

// AgentMeta provides monitors access to global agent metadata.  Putting this
// into a single interface allows easy expansion of metadata without breaking
// backwards-compatibility and without exposing global variables that monitors
// access.
type AgentMeta struct {
	Hostname                  string
	InternalMetricsSocketPath string
	CollectdConf              *config.CollectdConfig
}
