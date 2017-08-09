// Misc functions that facilitate templating of collectd config files.
package templating

import (
	"bytes"
	"net/url"
	"os"
	"path"
	"runtime"
	"text/template"

	log "github.com/sirupsen/logrus"

	"strings"
)

const pluginDir = "/usr/share/collectd"

func WriteConfFile(content, filePath string) bool {
	log.Debugf("Writing file %s", filePath)

	os.MkdirAll(path.Dir(filePath), 0755)
	f, err := os.Create(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  filePath,
		}).Error("failed to create/truncate collectd config file")
		return false
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  filePath,
		}).Error("failed to write collectd config file")
		return false
	}
	return true
}

func InjectTemplateFuncs(tmpl *template.Template) *template.Template {
	tmpl.Funcs(
		template.FuncMap{
			"EncodeDimsForPluginInstance": func(dims map[string]string) (string, error) {
				if dims == nil {
					return "", nil
				}

				var encoded []string
				for key, val := range dims {
					encoded = append(encoded, key+"="+val)
				}
				return strings.Join(encoded, ","), nil
			},
			"EncodeDimsAsQueryString": func(dims map[string]string) (string, error) {
				query := url.Values{}
				for k, v := range dims {
					query["sfxdim_"+k] = []string{v}
				}
				return "?" + query.Encode(), nil
			},
			"StringsJoin": strings.Join,
			"Globals": func() map[string]string {
				return map[string]string{
					"PluginRoot": pluginDir,
					"Platform":   runtime.GOOS,
				}
			},
			"RenderValue": func(templateText string, context map[string]interface{}, service interface{}) (string, error) {
				template, err := template.New("nested").Parse(templateText)
				if err != nil {
					return "", err
				}

				context["service"] = service

				out := bytes.Buffer{}
				err = template.Execute(&out, context)
				if err != nil {
					return "", err
				}

				return out.String(), nil
			},
		})
	return tmpl
}
