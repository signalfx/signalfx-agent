package config

import (
	"bytes"
	"fmt"
	"path"
	"text/template"
)

// RenderCollectdConf renders a collectd.conf config from the given app configuration.
func RenderCollectdConf(templatesDir string, appConfig *AppConfig) (string, error) {
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
		}).
		ParseGlob(path.Join(templatesDir, "*.tmpl")); err != nil {
		return "", fmt.Errorf("Failed to load templates: %s", err)
	} else if err := tmpl.Execute(&output, appConfig); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err)
	}

	return output.String(), nil
}
