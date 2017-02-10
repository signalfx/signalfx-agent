package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"runtime"
	"text/template"

	"github.com/signalfx/neo-agent/secrets"
)

// RenderCollectdConf renders a collectd.conf config from the given app configuration.
func RenderCollectdConf(pluginRoot string, templatesDir string, appConfig *AppConfig) (string, error) {
	if _, err := os.Stat(pluginRoot); os.IsNotExist(err) {
		return "", fmt.Errorf("plugin root directory %s does not exist", pluginRoot)
	}

	output := bytes.Buffer{}
	tmpl := template.New("collectd.conf.tmpl")

	if tmpl, err := tmpl.
		Funcs(template.FuncMap{
			"RenderTemplate": func(name string, data interface{}) (string, error) {
				buf := bytes.Buffer{}
				if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
					return "", err
				}
				return buf.String(), nil
			},
			"Globals": func() map[string]string {
				return map[string]string{
					"PluginRoot": pluginRoot,
					"Platform":   runtime.GOOS,
				}
			},
			"Secret": func(name string) (string, error) {
				// Try all secret keepers until either one succeeds or none
				// contain our secret.
				for _, keeper := range secrets.SecretKeepers {
					if val, err := keeper(name); err == nil {
						return val, nil
					}
				}

				return "", fmt.Errorf("unable to find secret for %s", name)
			},
		}).
		ParseGlob(path.Join(templatesDir, "*.tmpl")); err != nil {
		return "", fmt.Errorf("Failed to load templates: %s", err)
	} else if err := tmpl.Execute(&output, appConfig); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err)
	}

	return output.String(), nil
}
