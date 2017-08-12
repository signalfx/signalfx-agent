package templating

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// TemplateContext is what all of the collectd config/plugin templates get when
// executed.
type TemplateContext map[string]interface{}

// NewTemplateContext creates an uninitialized TemplateContext
func NewTemplateContext() TemplateContext {
	tc := TemplateContext(map[string]interface{}{})
	tc["_endpoints"] = make([]services.Endpoint, 0)
	tc["_dimensions"] = make(map[string]string)

	return tc
}

// SetEndpointInstances sets the full list of service instances in the context.
// There is no way to add them incrementally.
func (tc TemplateContext) SetEndpointInstances(endpoints []services.Endpoint) {
	ss := make([]services.Endpoint, len(endpoints), len(endpoints))
	for i := range endpoints {
		ss[i] = endpoints[i].(services.Endpoint)
	}
	tc["_endpoints"] = ss
}

// EndpointInstances returns whatever was set in SetEndpointInstances
func (tc TemplateContext) EndpointInstances() []services.Endpoint {
	return tc["_endpoints"].([]services.Endpoint)
}

// Endpoints returns a slice of the metadata for each service endpoint.  This metadata
// contains the same key/values used in discovery rules, so it will be
// consistent for the user.  Each endpoint map also includes the full template
// context for simplicity in the templates (so you don't have to remember to
// use $).
func (tc TemplateContext) Endpoints() []map[string]interface{} {
	ss := tc.EndpointInstances()
	out := make([]map[string]interface{}, len(ss), len(ss))

	for i := range tc.EndpointInstances() {
		out[i] = utils.MergeInterfaceMaps(
			// Merge in everything from the top level of the context so
			// that we have a unified namespace when looping over services
			// in templates.
			map[string]interface{}(tc),
			services.EndpointAsMap(ss[i]),
			// Merge dimensions from monitor config into service so that we
			// only have to reference one thing for dimensions.
			map[string]interface{}{
				"dimensions": utils.MergeStringMaps(tc.Dimensions(), ss[i].Dimensions()),
			})
	}
	return out
}

// SetDimensions is used to set the full set of dimensions that should be
// included with this plugin.
func (tc TemplateContext) SetDimensions(dims map[string]string) {
	tc["_dimensions"] = dims
}

// Dimensions is used by the plugin template to render the dimensions in the
// plugin instance so they will be sent to ingest.
func (tc TemplateContext) Dimensions() map[string]string {
	return tc["_dimensions"].(map[string]string)
}

// InjectConfigStruct facilitates converting a config struct that contains a lot
// of config for a monitor into a map.  It assumes that the struct has yaml
// tags saying how to convert the struct field names to yaml, which is used as
// the intermediate layer, so those names will be used in the context.
func (tc TemplateContext) InjectConfigStruct(configStruct interface{}) bool {
	contextConf, err := utils.ConvertToMapViaYAML(configStruct)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"endpoint": spew.Sdump(configStruct),
		}).Error("Could not convert config struct to map")
		return false
	}

	for k := range contextConf {
		tc[k] = contextConf[k]
	}
	return true
}
