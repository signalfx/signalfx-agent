package collectd

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/observers"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

type TemplateContext map[string]interface{}

func NewTemplateContext() TemplateContext {
	tc := TemplateContext(map[string]interface{}{})
	tc["services"] = make([]*observers.ServiceInstance, 0)
	tc["dimensions"] = make(map[string]string)

	return tc
}

func (tc TemplateContext) SetServices(services []*observers.ServiceInstance) {
	tc["services"] = services
}

func (tc TemplateContext) GetServices() []*observers.ServiceInstance {
	if ss, ok := tc["services"].([]*observers.ServiceInstance); ok {
		return ss
	}
	return nil
}

func (tc TemplateContext) SetDimensions(dims map[string]string) {
	tc["dimensions"] = dims
}

func (tc TemplateContext) GetDimensions() map[string]string {
	if dims, ok := tc["dimensions"].(map[string]string); ok {
		return dims
	}
	return nil
}

type BaseMonitor struct {
	Template *template.Template
	// The object that gets passed to the template execution
	Context TemplateContext
	// Where to write the plugin config to on the filesystem
	ConfigFilename string
}

func NewBaseMonitor(template *template.Template) *BaseMonitor {
	name := template.Name()
	templating.InjectTemplateFuncs(template)

	return &BaseMonitor{
		Template:       template,
		Context:        NewTemplateContext(),
		ConfigFilename: fmt.Sprintf("20-%s.%d.conf", name, getNextIdFor(name)),
	}
}

func (bm *BaseMonitor) SetConfigurationAndRun(conf config.MonitorConfig) bool {
	bm.Context["interval"] = conf.IntervalSeconds
	bm.Context["ingestURL"] = conf.IngestURL.String()
	bm.Context["accessToken"] = conf.SignalFxAccessToken
	//customTemplatePath := conf.OtherConfig["collectdTemplatePath"]

	bm.Context.SetDimensions(
		utils.MergeStringMaps(bm.Context.GetDimensions(), conf.ExtraDimensions))

	return bm.WriteConfigForPluginAndRestart()
}

func (bm *BaseMonitor) WriteConfigForPluginAndRestart() bool {
	pluginConfigText := bytes.Buffer{}
	err := bm.Template.Execute(&pluginConfigText, bm.Context)
	if err != nil {
		log.WithFields(log.Fields{
			"context":      bm.Context,
			"templateName": bm.Template.Name(),
			"error":        err,
		}).Error("Could not render collectd config file")
		return false
	}

	log.WithFields(log.Fields{
		"renderPath": bm.renderPath(),
		"context":    bm.Context,
	}).Debug("Writing collectd plugin config file")

	if !templating.WriteConfFile(pluginConfigText.String(), bm.renderPath()) {
		return false
	}

	CollectdSingleton.Restart()

	return true
}

func (bm *BaseMonitor) renderPath() string {
	return path.Join(managedConfigDir, bm.ConfigFilename)
}

func (bm *BaseMonitor) Shutdown() {
	os.Remove(bm.renderPath())
	CollectdSingleton.Restart()
}

var _ids = map[string]int{}

// Used to ensure unique filenames for distinct plugin templates that configure
// the same service/plugin
func getNextIdFor(name string) int {
	_ids[name] += 1
	return _ids[name]
}
