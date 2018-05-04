// +build windows

package collectd

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
)

// ConfigureMainCollectd should be called whenever the main collectd config in
// the agent has changed.  Restarts collectd if the config has changed.
func ConfigureMainCollectd(conf *config.CollectdConfig) error {
	return nil
}
