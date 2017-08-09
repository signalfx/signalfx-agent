// A custom collectd plugin monitor, for which you can specify your own config
// template and parameters.
package custom

import (
	"sync"
	"text/template"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
)

const monitorType = "collectd/custom"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &CustomMonitor{
			ServiceMonitorCore: *collectd.NewServiceMonitorCore(template.New("custom")),
		}
	}, &Config{})
}

type Config struct {
	config.MonitorConfig
	TemplateText    string
	TemplatePath    string
	TemplateContext map[string]interface{}
}

func (c *Config) Validate() bool {
	if (c.TemplateText == "") == (c.TemplatePath == "") {
		log.WithFields(log.Fields{
			"monitorType": monitorType,
			"config":      *c,
		}).Error("Exactly one of either templateText or templatePath must be set")
		return false
	}
	return c.getTemplate() != nil
}

func (c *Config) getTemplate() *template.Template {
	var templateText string
	if c.TemplatePath != "" {
		source, path, err := c.MetaStore.GetSourceAndPath(c.TemplatePath)
		if err != nil {
			log.WithFields(log.Fields{
				"error":        err,
				"templatePath": c.TemplatePath,
			}).Error("Template path type is unrecognized")
			return nil
		}
		kv, err := source.Get(path)
		if err != nil {
			log.WithFields(log.Fields{
				"error":        err,
				"templatePath": c.TemplatePath,
			}).Error("Could not access template path")
			return nil
		}
		templateText = string(kv.Value)
	} else {
		templateText = c.TemplateText
	}
	return templateFromText(templateText)
}

func templateFromText(templateText string) *template.Template {
	template, err := templating.InjectTemplateFuncs(template.New("custom")).Parse(templateText)
	if err != nil {
		log.WithFields(log.Fields{
			"monitorType":  monitorType,
			"templateText": templateText,
		}).Error("Template text failed to parse!")
		return nil
	}
	return template
}

type CustomMonitor struct {
	collectd.ServiceMonitorCore
	// Used to stop watching if we are loading the template from a path
	stopWatchCh chan struct{}
	lock        sync.Mutex
}

func (cm *CustomMonitor) Configure(conf *Config) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	for k, v := range conf.TemplateContext {
		cm.Context[k] = v
	}
	cm.Template = conf.getTemplate()
	if cm.Template == nil {
		return false
	}

	if conf.TemplatePath != "" {
		return cm.watchTemplatePath(conf)
	}

	return cm.SetConfigurationAndRun(conf.MonitorConfig)
}

func (cm *CustomMonitor) watchTemplatePath(conf *Config) bool {
	templateLoads, stopWatch, err := conf.MetaStore.WatchPath(conf.TemplatePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"templatePath": conf.TemplatePath,
		}).Error("Could not watch template path for custom collectd monitor")
		return false
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

				cm.Template = templateFromText(string(templateKV.Value))
				if cm.Template == nil {
					continue
				}
				cm.SetConfigurationAndRun(conf.MonitorConfig)

				cm.lock.Unlock()
			}
		}
	}()
	return true
}

func (cm *CustomMonitor) Shutdown() {
	if cm.stopWatchCh != nil {
		cm.stopWatchCh <- struct{}{}
	}
}
