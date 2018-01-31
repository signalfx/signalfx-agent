package collectd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	"github.com/signalfx/neo-agent/monitors/types"
	log "github.com/sirupsen/logrus"
)

// MonitorCore contains common data/logic for collectd monitors, mainly
// stuff related to templating of the plugin config files.  This should
// generally not be used directly, but rather one of the structs that embeds
// this: StaticMonitorCore or ServiceMonitorCore.
type MonitorCore struct {
	Template *template.Template
	Output   types.Output
	// Where to write the plugin config to on the filesystem
	configFilename string
	config         config.MonitorCustomConfig
	isRunning      bool
	monitorID      types.MonitorID
	lock           sync.Mutex
	UsesGenericJMX bool
}

// NewMonitorCore creates a new initialized but unconfigured MonitorCore with
// the given template.
func NewMonitorCore(template *template.Template) *MonitorCore {
	return &MonitorCore{
		Template:  template,
		isRunning: false,
	}
}

// Init generates a unique file name for each distinct monitor instance
func (bm *MonitorCore) Init() error {
	templating.InjectTemplateFuncs(bm.Template)

	return nil
}

// SetConfigurationAndRun sets the configuration to be used when rendering
// templates, and writes config before queueing a collectd restart.
func (bm *MonitorCore) SetConfigurationAndRun(conf config.MonitorCustomConfig) error {
	bm.lock.Lock()
	defer bm.lock.Unlock()

	bm.config = conf
	bm.monitorID = conf.MonitorConfigCore().MonitorID

	bm.configFilename = fmt.Sprintf("20-%s.%s.conf", bm.Template.Name(), string(bm.monitorID))

	if err := bm.WriteConfigForPlugin(); err != nil {
		return err
	}
	return bm.SetConfiguration(conf)
}

// SetConfiguration adds various fields from the config to the template context
// but does not render the config.
func (bm *MonitorCore) SetConfiguration(conf config.MonitorCustomConfig) error {
	if err := Instance().ConfigureFromMonitor(bm.monitorID, bm.Output, bm.UsesGenericJMX); err != nil {
		return err
	}

	bm.isRunning = true
	return nil
}

// WriteConfigForPlugin will render the config template to the filesystem and
// queue a collectd restart
func (bm *MonitorCore) WriteConfigForPlugin() error {
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
		log.WithFields(log.Fields{
			"error": err,
			"path":  bm.renderPath(),
		}).Error("Could not render collectd plugin config")
		return err
	}

	return nil
}

func (bm *MonitorCore) renderPath() string {
	return filepath.Join(managedConfigDir, bm.configFilename)
}

// Shutdown removes the config file and restarts collectd
func (bm *MonitorCore) Shutdown() {
	log.WithFields(log.Fields{
		"path": bm.renderPath(),
	}).Debug("Removing collectd plugin config")

	os.Remove(bm.renderPath())
	Instance().MonitorDidShutdown(bm.monitorID)
}
