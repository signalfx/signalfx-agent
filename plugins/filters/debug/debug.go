package debug

import (
	"encoding/json"
	"log"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

// Filter prints the input and passes
type Filter struct {
	plugins.Plugin
}

func init() {
	plugins.Register("filters/debug", NewFilter)
}

// NewFilter creates a new instance
func NewFilter(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, config)
	if err != nil {
		return nil, err
	}

	return &Filter{plugin}, nil
}

// Map prints the input
func (d *Filter) Map(sis services.Instances) (services.Instances, error) {
	if bytes, err := json.MarshalIndent(sis, "", "  "); err == nil {
		log.Printf("Debug:\n%s", string(bytes))
	}
	return sis, nil
}
