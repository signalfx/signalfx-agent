package collectd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	"strings"
)

// WriteConfFile writes a file to the given filePath, ensuring that the
// containing directory exists.
func WriteConfFile(content, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return errors.Wrapf(err, "failed to create collectd config dir at %s", filepath.Dir(filePath))
	}

	f, err := os.Create(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create/truncate collectd config file at %s", filePath)
	}
	defer f.Close()

	// Lock the file down since it could contain credentials
	if err := f.Chmod(0600); err != nil {
		return errors.Wrapf(err, "failed to restrict permissions on collectd config file at %s", filePath)
	}

	_, err = f.Write([]byte(content))
	if err != nil {
		return errors.Wrapf(err, "Failed to write collectd config file at %s", filePath)
	}

	log.Debugf("Wrote file %s", filePath)

	return nil
}

// InjectTemplateFuncs injects some helper functions into our templates.
func InjectTemplateFuncs(tmpl *template.Template) *template.Template {
	tmpl.Funcs(
		template.FuncMap{
			// Global variables available in all templates
			"Globals": func() map[string]string {
				return map[string]string{
					"Platform": runtime.GOOS,
				}
			},
			"pluginRoot": func() string {
				return Instance().PluginDir()
			},
			"withDefault": func(value interface{}, def interface{}) interface{} {
				v := reflect.ValueOf(value)
				switch v.Kind() {
				case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
					if v.Len() == 0 {
						return def
					}
				case reflect.Ptr:
					if v.IsNil() {
						return def
					}
				case reflect.Invalid:
					return def
				default:
					return value
				}
				return value
			},
			// Makes a slice of the given values
			"sliceOf": func(values ...interface{}) []interface{} {
				return values
			},
			// Encodes dimensions in our "key=value,..." encoding that gets put
			// in the collectd plugin_instance
			"encodeDimsForPluginInstance": func(dims ...map[string]string) (string, error) {
				var encoded []string
				for i := range dims {
					for key, val := range dims[i] {
						encoded = append(encoded, key+"="+val)
					}
				}
				return strings.Join(encoded, ","), nil
			},
			// Encode dimensions for use in an ingest URL
			"encodeDimsAsQueryString": func(dims map[string]string) (string, error) {
				query := url.Values{}
				for k, v := range dims {
					query["sfxdim_"+k] = []string{v}
				}
				return "?" + query.Encode(), nil
			},
			"stringsJoin": func(ss []string, joiner string) string {
				return strings.Join(ss, joiner)
			},
			"stripTrailingSlash": func(s string) string {
				return strings.TrimSuffix(s, "/")
			},
			// Tells whether the key is present in the context map.  Says
			// nothing about whether it is a zero-value or not.
			"hasKey": func(key string, context map[string]interface{}) bool {
				_, ok := context[key]
				return ok
			},
			"merge":           utils.MergeInterfaceMaps,
			"mergeStringMaps": utils.MergeStringMaps,
			"toBool": func(v interface{}) (string, error) {
				if v == nil {
					return "false", nil
				}
				if b, ok := v.(*bool); ok {
					if *b {
						return "true", nil
					}
					return "false", nil
				}
				if b, ok := v.(bool); ok {
					if b {
						return "true", nil
					}
					return "false", nil
				}
				return "", fmt.Errorf("Value %#v cannot be converted to bool", v)
			},
			"toMap": utils.ConvertToMapViaYAML,
			"toServiceID": func(s string) services.ID {
				return services.ID(s)
			},
			"toStringMap": utils.InterfaceMapToStringMap,
			"spew": func(obj interface{}) string {
				return spew.Sdump(obj)
			},
			// Renders a subtemplate using the provided context, and optionally
			// a service, which will be added to the context as "service"
			"renderValue": func(templateText string, context interface{}) (string, error) {
				if templateText == "" {
					return "", nil
				}

				template, err := template.New("nested").Parse(templateText)
				if err != nil {
					return "", err
				}

				out := bytes.Buffer{}
				err = template.Option("missingkey=error").Execute(&out, context)
				if err != nil {
					log.WithFields(log.Fields{
						"template": templateText,
						"error":    err,
						"context":  spew.Sdump(context),
					}).Error("Could not render nested config template")
					return "", err
				}

				return out.String(), nil
			},
		})
	return tmpl
}
