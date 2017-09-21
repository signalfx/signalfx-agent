package collectd

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// BaseMonitor contains common data/logic for collectd monitors, mainly
// stuff related to templating of the plugin config files.  This should
// generally not be used directly, but rather one of the structs that embeds
// this: StaticMonitorCore or ServiceMonitorCore.
type BaseMonitor struct {
	Template *template.Template
	// The object that gets passed to the template execution
	Context templating.TemplateContext
	// Where to write the plugin config to on the filesystem
	ConfigFilename string
	isRunning      bool
	monitorID      types.MonitorID
	lock           sync.Mutex
}

// NewBaseMonitor creates a new initialized but unconfigured BaseMonitor with
// the given template.
func NewBaseMonitor(template *template.Template) *BaseMonitor {
	name := template.Name()
	templating.InjectTemplateFuncs(template)

	return &BaseMonitor{
		Template:       template,
		Context:        templating.NewTemplateContext(),
		ConfigFilename: fmt.Sprintf("20-%s.%d.conf", name, getNextIDFor(name)),
		isRunning:      false,
	}
}

// SetConfiguration adds various fields from the config to the template context
// but does not render the config.
func (bm *BaseMonitor) SetConfiguration(conf *config.MonitorConfig) bool {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.Context["IntervalSeconds"] = conf.IntervalSeconds
	bm.Context["IngestURL"] = conf.IngestURL.String()
	bm.Context["SignalFxAccessToken"] = conf.SignalFxAccessToken

	bm.Context.SetDimensions(
		utils.MergeStringMaps(bm.Context.Dimensions(), conf.ExtraDimensions))
	log.Debugf("Setting dimensions on %s to: %s", conf.Type, utils.MergeStringMaps(bm.Context.Dimensions(), conf.ExtraDimensions))

	bm.monitorID = conf.ID
	if !Instance().ConfigureFromMonitor(conf.ID, conf.CollectdConf) {
		return false
	}

	return true
}

// WriteConfigForPluginAndRestart will render the config template to the
// filesystem and queue a collectd restart
func (bm *BaseMonitor) WriteConfigForPluginAndRestart() bool {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	pluginConfigText := bytes.Buffer{}

	err := bm.Template.Execute(&pluginConfigText, bm.Context)
	if err != nil {
		log.WithFields(log.Fields{
			"context":      spew.Sdump(bm.Context),
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

	Instance().Restart()

	bm.isRunning = true

	return true
}

func (bm *BaseMonitor) renderPath() string {
	return path.Join(managedConfigDir, bm.ConfigFilename)
}

// Shutdown removes the config file and restarts collectd
func (bm *BaseMonitor) Shutdown() {
	os.Remove(bm.renderPath())
	Instance().MonitorDidShutdown(bm.monitorID)
}

var _ids = map[string]int{}

// Used to ensure unique filenames for distinct plugin templates that configure
// the same service/plugin
func getNextIDFor(name string) int {
	_ids[name]++
	return _ids[name]
}
