package collectd

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/core/services"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	log "github.com/sirupsen/logrus"
)

// BaseMonitor contains common data/logic for collectd monitors, mainly
// stuff related to templating of the plugin config files.  This should
// generally not be used directly, but rather one of the structs that embeds
// this: StaticMonitorCore or ServiceMonitorCore.
type BaseMonitor struct {
	Template *template.Template
	// The object that gets passed to the template execution
	Context struct {
		C         config.MonitorCustomConfig
		Endpoints []services.Endpoint
	}
	// Where to write the plugin config to on the filesystem
	configFilename string
	isRunning      bool
	monitorID      types.MonitorID
	lock           sync.Mutex
	DPs            chan<- *datapoint.Datapoint
	Events         chan<- *event.Event
}

// NewBaseMonitor creates a new initialized but unconfigured BaseMonitor with
// the given template.
func NewBaseMonitor(template *template.Template) *BaseMonitor {
	return &BaseMonitor{
		Template:  template,
		isRunning: false,
	}
}

func (bm *BaseMonitor) Init() error {
	name := bm.Template.Name()
	bm.configFilename = fmt.Sprintf("20-%s.%d.conf", name, getNextIDFor(name))
	templating.InjectTemplateFuncs(bm.Template)

	return nil
}

// SetConfiguration adds various fields from the config to the template context
// but does not render the config.
func (bm *BaseMonitor) SetConfiguration(conf config.MonitorCustomConfig) bool {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.Context.C = conf

	bm.monitorID = conf.CoreConfig().ID
	if !Instance().ConfigureFromMonitor(conf.CoreConfig().ID, conf.CoreConfig().CollectdConf, bm.DPs, bm.Events) {
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
	return path.Join(managedConfigDir, bm.configFilename)
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
