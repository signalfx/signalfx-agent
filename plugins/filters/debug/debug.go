package debug

import (
	"log"

	"encoding/json"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

// DebugFilter prints the input and passes
type DebugFilter struct {
	plugins.Plugin
}

// NewDebugFilter creates a new instance
func NewDebugFilter(name string, config *viper.Viper) (*DebugFilter, error) {
	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}

	return &DebugFilter{plugin}, nil
}

// Map prints the input
func (d *DebugFilter) Map(sis services.ServiceInstances) (services.ServiceInstances, error) {
	for _, s := range sis {
		if bytes, err := json.MarshalIndent(s, "", "  "); err == nil {
			log.Print(string(bytes))
		}
	}

	return sis, nil
}
