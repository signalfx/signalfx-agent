package debug

import (
	"encoding/json"
	"log"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
)

const (
	pluginType = "filters/debug"
)

// Filter prints the input and passes
type Filter struct {}

func init() {
	plugins.Register(pluginType, func() interface{} { return &Filter{} })
}

// Map prints the input
func (d *Filter) Map(sis services.Instances) (services.Instances, error) {
	if bytes, err := json.MarshalIndent(sis, "", "  "); err == nil {
		log.Printf("Debug:\n%s", string(bytes))
	}
	return sis, nil
}
