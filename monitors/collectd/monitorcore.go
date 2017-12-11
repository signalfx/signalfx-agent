package collectd

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	log "github.com/sirupsen/logrus"
)

// MonitorCore contains common data/logic for collectd monitors, mainly
// stuff related to templating of the plugin config files.  This should
// generally not be used directly, but rather one of the structs that embeds
// this: StaticMonitorCore or ServiceMonitorCore.
type MonitorCore struct {
	Template *template.Template
	// Where to write the plugin config to on the filesystem
	configFilename string
	config         config.MonitorCustomConfig
	isRunning      bool
	monitorID      monitors.MonitorID
	lock           sync.Mutex
	DPs            chan<- *datapoint.Datapoint
	Events         chan<- *event.Event
	UsesGenericJMX bool
}

// NewMonitorCore creates a new initialized but unconfigured MonitorCore with
// the given template.
func NewMonitorCore(id monitors.MonitorID, template *template.Template) *MonitorCore {
	return &MonitorCore{
		Template:  template,
		isRunning: false,
		monitorID: id,
	}
}

// Init generates a unique file name for each distinct monitor instance
func (bm *MonitorCore) Init() error {
	name := bm.Template.Name()
	bm.configFilename = fmt.Sprintf("20-%s.%s.conf", name, string(bm.monitorID))
	templating.InjectTemplateFuncs(bm.Template)

	return nil
}

// SetConfigurationAndRun sets the configuration to be used when rendering
// templates, and writes config before queueing a collectd restart.
func (bm *MonitorCore) SetConfigurationAndRun(conf config.MonitorCustomConfig) error {
	if err := bm.SetConfiguration(conf); err != nil {
		return err
	}
	return bm.WriteConfigForPluginAndRestart()
}

// SetConfiguration adds various fields from the config to the template context
// but does not render the config.
func (bm *MonitorCore) SetConfiguration(conf config.MonitorCustomConfig) error {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.config = conf

	return Instance().ConfigureFromMonitor(bm.monitorID, conf.CoreConfig().CollectdConf, bm.DPs, bm.Events, bm.UsesGenericJMX)
}

// WriteConfigForPluginAndRestart will render the config template to the
// filesystem and queue a collectd restart
func (bm *MonitorCore) WriteConfigForPluginAndRestart() error {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	pluginConfigText := bytes.Buffer{}

	err := bm.Template.Execute(&pluginConfigText, bm.config)
	if err != nil {
		return errors.Wrapf(err, "Could not render collectd config file for %s.  Context was %#v",
			bm.Template.Name(), bm.config)
	}

	log.WithFields(log.Fields{
		"renderPath": bm.renderPath(),
		"context":    bm.config,
	}).Debug("Writing collectd plugin config file")

	if err := templating.WriteConfFile(pluginConfigText.String(), bm.renderPath()); err != nil {
		return err
	}

	Instance().Restart()

	bm.isRunning = true

	return nil
}

func (bm *MonitorCore) renderPath() string {
	return path.Join(managedConfigDir, bm.configFilename)
}

// Shutdown removes the config file and restarts collectd
func (bm *MonitorCore) Shutdown() {
	os.Remove(bm.renderPath())
	Instance().MonitorDidShutdown(bm.monitorID)
}
