// Package custom contains a custom collectd plugin monitor, for which you can
// specify your own config template and parameters.
package custom

import (
	"fmt"
	"sync"
	"text/template"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
	log "github.com/sirupsen/logrus"
)

const monitorType = "collectd/custom"

func init() {
	monitors.Register(monitorType, func(id monitors.MonitorID) interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(id, template.New("custom")),
		}
	}, &Config{})
}

// Config is the configuration for the collectd custom monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`

	Host string  `yaml:"host"`
	Port uint16  `yaml:"port"`
	Name *string `yaml:"name"`

	TemplateText string `yaml:"templateText"`
	TemplatePath string `yaml:"templatePath"`
}

// Validate will check the config that is specific to this monitor
func (c *Config) Validate() error {
	if (c.TemplateText == "") == (c.TemplatePath == "") {
		return errors.New("Exactly one of either templateText or templatePath must be set")
	}
	if _, err := c.getTemplate(); err != nil {
		return err
	}
	return nil
}

func (c *Config) getTemplate() (*template.Template, error) {
	var templateText string
	if c.TemplatePath != "" {
		source, path, err := c.MetaStore.GetSourceAndPath(c.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("Template path type '%s' is unrecognized: %s", c.TemplatePath, err)
		}
		kv, err := source.Get(path)
		if err != nil {
			return nil, fmt.Errorf("Could not access template path %s: %s", c.TemplatePath, err)
		}
		templateText = string(kv.Value)
	} else {
		templateText = c.TemplateText
	}
	return templateFromText(templateText)
}

func templateFromText(templateText string) (*template.Template, error) {
	template, err := templating.InjectTemplateFuncs(template.New("custom")).Parse(templateText)
	if err != nil {
		return nil, errors.Wrapf(err, "Template text failed to parse: \n%s", templateText)
	}
	return template, nil
}

// Monitor is the core monitor object that gets instantiated by the agent
type Monitor struct {
	collectd.MonitorCore
	// Used to stop watching if we are loading the template from a path
	stopWatchCh chan struct{}
	lock        sync.Mutex
}

// Configure will render the custom collectd config and queue a collectd
// restart.
func (cm *Monitor) Configure(conf *Config) error {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	var err error
	cm.Template, err = conf.getTemplate()
	if err != nil {
		return err
	}

	if conf.TemplatePath != "" {
		return cm.watchTemplatePath(conf)
	}

	return cm.SetConfigurationAndRun(conf)
}

func (cm *Monitor) watchTemplatePath(conf *Config) error {
	templateLoads, stopWatch, err := conf.MetaStore.WatchPath(conf.TemplatePath)
	if err != nil {
		return errors.Wrapf(err, "Could not watch template path %s for custom collectd monitor", conf.TemplatePath)
	}

	cm.stopWatchCh = make(chan struct{})

	go func() {
		for {
			select {
			case <-cm.stopWatchCh:
				stopWatch()
				return
			case templateKV := <-templateLoads:
				cm.lock.Lock()

				cm.Template, err = templateFromText(string(templateKV.Value))
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
						"text":  templateKV.Value,
					}).Error("Could not load template from text")
					continue
				}
				cm.SetConfigurationAndRun(conf)

				cm.lock.Unlock()
			}
		}
	}()
	return nil
}

// Shutdown stops the file watching if using a template path
func (cm *Monitor) Shutdown() {
	if cm.stopWatchCh != nil {
		cm.stopWatchCh <- struct{}{}
	}
}
