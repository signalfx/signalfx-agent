// Package custom contains a custom collectd plugin monitor, for which you can
// specify your own config template and parameters.
package custom

import (
	"text/template"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/collectd"
	"github.com/signalfx/neo-agent/monitors/collectd/templating"
)

const monitorType = "collectd/custom"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(template.New("custom")),
		}
	}, &Config{})
}

// Config is the configuration for the collectd custom monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
	Name string `yaml:"name"`

	Template  string   `yaml:"template"`
	Templates []string `yaml:"templates"`
}

func (c *Config) AllTemplates() []string {
	templates := c.Templates
	if c.Template != "" {
		templates = append(templates, c.Template)
	}
	return templates
}

// Validate will check the config that is specific to this monitor
func (c *Config) Validate() error {
	for _, templateText := range c.AllTemplates() {
		if _, err := templateFromText(templateText); err != nil {
			return err
		}
	}

	if len(c.ExtraDimensions) > 0 {
		return errors.New("Collectd custom template monitors cannot have " +
			"extraDimensions set because there is no generic way to correlate " +
			"datapoints from those plugins to their configuration")
	}
	return nil
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
}

// Configure will render the custom collectd config and queue a collectd
// restart.
func (cm *Monitor) Configure(conf *Config) error {
	templateTextConcatenated := ""
	for _, text := range conf.AllTemplates() {
		templateTextConcatenated += "\n" + text
	}

	// Allow blank template text so that we have a standard config item that
	// configured the monitor with all of the templates in a possibly
	// non-existant legacy collectd managed_config dir.
	if templateTextConcatenated == "" {
		return nil
	}

	var err error
	cm.Template, err = templateFromText(templateTextConcatenated)
	if err != nil {
		return err
	}

	cm.MonitorCore.NoMonitorID = true

	return cm.SetConfigurationAndRun(conf)
}
