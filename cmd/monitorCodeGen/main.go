package main

import (
	"bytes"
	"fmt"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

var tmpl = `// DO NOT EDIT. This file is auto-generated.

package {{.goPackage}}

import "github.com/signalfx/signalfx-agent/internal/monitors"

{{with .groupMetricsMap}}
const (
{{- range . }}
	{{(printf "group.%s" .Name) | formatVariable}} = "{{.Name}}"
{{- end}}
)
{{end}}

var groupSet = map[string]bool {
{{- range .groupMetricsMap}}
	{{(printf "group.%s" .Name) | formatVariable}}: true,
{{- end}}
}

{{with .metrics}}
const (
{{- range .}}
	{{.Name | formatVariable}} = "{{.Name}}"
{{- with .Alias}}
	{{. | formatVariable}} = "{{.}}"
{{- end}}
{{- end}}
)
{{end}}

var metricSet = map[string]bool {
{{- range .metrics}}
	{{.Name | formatVariable}}: true,
{{- end}}
}

var includedMetrics = map[string]bool {
{{- range .metrics}}
{{- if .Included}}
	{{.Name | formatVariable}}: true,
{{- with .Alias}}
	{{. | formatVariable}}: true,
{{- end}}
{{- end}}
{{- end}}
}

var groupMetricsMap = map[string][]string {
{{- range $group, $metrics := .groupMetricsMap}}
	{{(printf "group.%s" $group) | formatVariable}}: []string {
		{{- range $metrics}}
		{{. | formatVariable}},
		{{- end}}
	},
{{- end}}
}

{{/* var {{namespaceMetadata .MonitorType "monitorMetadata" $.monitors}} */}}

{{range .monitors}}
var {{if gt (len $.monitors) 1 -}}
{{- (printf "%s%s" .MonitorType "monitorMetadata") | formatVariable -}}
{{else -}}
monitorMetadata{{end}} = monitors.Metadata{
	MonitorType: "{{.MonitorType}}",
	IncludedMetrics: includedMetrics,
	Metrics: metricSet,
	MetricsExhaustive: {{.MetricsExhaustive}},
	Groups: groupSet,
	GroupMetricsMap: groupMetricsMap,
	SendAll: {{ .SendAll }},
}
{{end}}
`

// shouldRegenerate determines whether the metadata file needs regenerated based on its existence and timestamps.
func shouldRegenerate(pkg *monitors.PackageMetadata) (bool, error) {
	generatedMetadata := filepath.Join(pkg.PackagePath, "generatedMetadata.go")

	var generatorStat os.FileInfo
	var statMetadataYaml os.FileInfo

	if path, err := os.Executable(); err != nil {
		return false, err
	} else if generatorStat, err = os.Stat(path); err != nil {
		return false, err
	} else if statMetadataYaml, err = os.Stat(pkg.Path); err != nil {
		return false, fmt.Errorf("unable to stat %s", pkg.Path)
	}

	statMetadataGenerated, err := os.Stat(generatedMetadata)

	if err != nil {
		// generatedMetadata.go does not exist.
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("unable to stat %s", generatedMetadata)
	}

	// There's an existing generatedMetadata.go so check timestamps.
	return statMetadataYaml.ModTime().After(statMetadataGenerated.ModTime()) ||
		generatorStat.ModTime().After(statMetadataGenerated.ModTime()), nil
}

func generate() error {
	pkgs, err := monitors.CollectMetadata("internal/monitors")

	if err != nil {
		return err
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{
		"formatVariable": formatVariable,
	}).Parse(tmpl)

	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if regenerate, err := shouldRegenerate(&pkg); err != nil {
			return err
		} else if !regenerate {
			continue
		}

		// Go package name can be overriden but default to the directory name.
		var goPackage string
		if pkg.GoPackage != nil {
			goPackage = *pkg.GoPackage
		} else {
			goPackage = filepath.Base(pkg.PackagePath)
		}

		writer := &bytes.Buffer{}
		groupMetricsMap := map[string][]string{}
		metrics := map[string]monitors.MetricMetadata{}

		for _, mon := range pkg.Monitors {
			for _, metric := range mon.Metrics {
				if existingMetric, ok := metrics[metric.Name]; ok {
					if existingMetric.Type != metric.Type {
						return fmt.Errorf("existing metric %v does not have the same metric type as %v", existingMetric,
							metric)
					}
				} else {
					metrics[metric.Name] = metric
				}

				if metric.Group != nil {
					metrics := []string{metric.Name}
					if metric.Alias != "" {
						metrics = append(metrics, metric.Alias)
					}
					groupMetricsMap[*metric.Group] = append(groupMetricsMap[*metric.Group], metrics...)
				}
			}
		}

		if err := tmpl.Execute(writer, map[string]interface{}{
			"metrics":         metrics,
			"monitors":        pkg.Monitors,
			"goPackage":       goPackage,
			"groupMetricsMap": groupMetricsMap,
		}); err != nil {
			return fmt.Errorf("failed executing template for %s: %s", pkg.Path, err)
		}

		if err := ioutil.WriteFile(filepath.Join(pkg.PackagePath, "generatedMetadata.go"), writer.Bytes(),
			0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := generate(); err != nil {
		log.Fatal(err)
	}
}
