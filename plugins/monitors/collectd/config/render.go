package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"text/template"

	"path"

	"strings"

	"github.com/docker/libkv/store"
	"github.com/signalfx/neo-agent/secrets"
)

// loadTemplates loads templates from the stores data
func loadTemplates(t *template.Template, files []*store.KVPair) error {
	for _, file := range files {
		name := path.Base(file.Key)
		if !strings.HasSuffix(name, ".tmpl") {
			continue
		}

		var tmpl *template.Template

		if t.Name() == name {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}

		if _, err := tmpl.Parse(string(file.Value)); err != nil {
			return err
		}
	}

	return nil
}

// RenderCollectdConf renders a collectd.conf config from the given app configuration.
func RenderCollectdConf(pluginRoot string, builtins, overrides []*store.KVPair, appConfig *AppConfig) (string, error) {
	if _, err := os.Stat(pluginRoot); os.IsNotExist(err) {
		return "", fmt.Errorf("plugin root directory %s does not exist", pluginRoot)
	}

	output := bytes.Buffer{}
	tmpl := template.New("collectd.conf.tmpl")
	tmpl.Funcs(
		template.FuncMap{
			"RenderTemplate": func(templateFile string, data interface{}) (string, error) {
				buf := bytes.Buffer{}

				if err := tmpl.ExecuteTemplate(&buf, templateFile, data); err != nil {
					return "", err
				}
				return buf.String(), nil
			},
			"RenderPlugin": func(plugin *Plugin) (string, error) {
				buf := bytes.Buffer{}
				if plugin.Template != "" {
					tmp, err := template.New("_sfx_render_template").Parse(plugin.Template)
					if err != nil {
						return "", err
					}

					if err := tmp.Execute(&buf, plugin); err != nil {
						return "", err
					}

					return buf.String(), nil
				} else if plugin.TemplateFile != "" {
					if err := tmpl.ExecuteTemplate(&buf, plugin.TemplateFile, plugin); err != nil {
						return "", err
					}
					return buf.String(), nil
				} else {
					return "", errors.New("neither Template nor TemplateFile is set")
				}
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
		})

	log.Printf("loading %d builtin templates", len(builtins))

	if err := loadTemplates(tmpl, builtins); err != nil {
		return "", fmt.Errorf("failed to load builtin templates: %s", err)
	}

	log.Printf("loading %d override templates", len(overrides))

	if err := loadTemplates(tmpl, overrides); err != nil {
		return "", fmt.Errorf("failed to load override templates: %s", err)
	}

	if err := tmpl.Execute(&output, appConfig); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err)
	}

	return output.String(), nil
}
