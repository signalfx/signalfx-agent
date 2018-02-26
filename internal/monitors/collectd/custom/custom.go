// Package custom contains a custom collectd plugin monitor, for which you can
// specify your own config template and parameters.
package custom

import (
	"text/template"

	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/custom"

// MONITOR(collectd/custom): This monitor lets you provide custom collectd
// configuration to be run by the managed collectd instance.  The provided
// config is parsed as a Go template, so if you provide a `discoveryRule` to
// automatically discover service endpoints, the agent will create a separate
// copy of the config for each endpoint discovered and fill in the `{{.Host}}`
// and `{{.Port}}` parameters in the provided config with those of the
// discovered endpoint.  If you do not provide a discovery rule, the provided
// config will be run immediately with collectd.
//
// You can provide configuration for as many plugins as you want in a single
// instance of this monitor configuration.
//
// You can also use your own Python plugins in conjunction with the
// `ModulePath` option in
// [collectd-python](https://collectd.org/documentation/manpages/collectd-python.5.shtml).
// If your Python plugin has dependencies of its own, you can specify the path
// to them by specifying multiple `ModulePath` options with those paths.
//
// Here is an example of a configuration with a custom Python plugin:
//
// ```yaml
//   - type: collectd/custom
//     discoveryRule: containerImage =~ "myservice"
//     template: |
//       LoadPlugin "python"
//       <Plugin python>
//         ModulePath "/usr/lib/python2.7/dist-packages/health_checker"
//         Import "health_checker"
//         <Module health_checker>
//           URL "http://{{.Host}}:{{.Port}}"
//           JSONKey "isRunning"
//           JSONVal "1"
//         </Module>
//       </Plugin>
// ```

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

	// This should generally not be set manually, but will be filled in by the
	// agent if using service discovery. It can be accessed in the provided
	// config template with `{{.Host}}`.  It will be set to the hostname or IP
	// address of the discovered service. If you aren't using service
	// discovery, you can just hardcode the host/port in the config template
	// and ignore these fields.
	Host string `yaml:"host"`
	// This should generally not be set manually, but will be filled in by the
	// agent if using service discovery. It can be accessed in the provided
	// config template with `{{.Port}}`.  It will be set to the port of the
	// discovered service, if it is a TCP/UDP endpoint.
	Port uint16 `yaml:"port"`
	// This should generally not be set manually, but will be filled in by the
	// agent if using service discovery. It can be accessed in the provided
	// config template with `{{.Name}}`.  It will be set to the name that the
	// observer creates for the endpoint upon discovery.  You can generally
	// ignore this field.
	Name string `yaml:"name"`

	// A config template for collectd.  You can include as many plugin blocks
	// as you want in this value.  It is rendered as a standard Go template, so
	// be mindful of the strings `{{` and `}}`.
	Template string `yaml:"template"`
	// A list of templates, but otherwise equivalent to the above `template`
	// option.  This enables you to have a single directory with collectd
	// configuration files and load them all by using a globbed remote config
	// value:
	Templates []string `yaml:"templates"`
}

func (c *Config) allTemplates() []string {
	templates := c.Templates
	if c.Template != "" {
		templates = append(templates, c.Template)
	}
	return templates
}

// Validate will check the config that is specific to this monitor
func (c *Config) Validate() error {
	for _, templateText := range c.allTemplates() {
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
	template, err := collectd.InjectTemplateFuncs(template.New("custom")).Parse(templateText)
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
	for _, text := range conf.allTemplates() {
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
