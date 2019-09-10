package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/signalfx/signalfx-agent/internal/selfdescribe"
	log "github.com/sirupsen/logrus"
)

const (
	genMetadata               = "genmetadata.go"
	generatedMetadataTemplate = "genmetadata.tmpl"
)

func buildOutputPath(pkg *selfdescribe.PackageMetadata) string {
	outputDir := pkg.PackagePath
	outputPackage := strings.TrimSpace(pkg.PackageDir)
	if outputPackage != "" {
		outputDir = path.Join(pkg.PackagePath, outputPackage)
	}
	return path.Join(outputDir, genMetadata)
}

// shouldRegenerate determines whether the metadata file needs regenerated based on its existence and timestamps.
func shouldRegenerate(pkg *selfdescribe.PackageMetadata, outputFile string) (bool, error) {
	var generatorStat os.FileInfo
	var statMetadataYaml os.FileInfo

	if path, err := os.Executable(); err != nil {
		return false, err
	} else if generatorStat, err = os.Stat(path); err != nil {
		return false, err
	} else if statMetadataYaml, err = os.Stat(pkg.Path); err != nil {
		return false, fmt.Errorf("unable to stat %s", pkg.Path)
	}

	statMetadataGenerated, err := os.Stat(outputFile)

	if err != nil {
		// generatedMetadata.go does not exist.
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("unable to stat %s", outputFile)
	}

	// There's an existing generatedMetadata.go so check timestamps.
	return statMetadataYaml.ModTime().After(statMetadataGenerated.ModTime()) ||
		generatorStat.ModTime().After(statMetadataGenerated.ModTime()), nil
}

func generate(templateFile string, force bool) error {
	pkgs, err := selfdescribe.CollectMetadata("internal/monitors")

	if err != nil {
		return err
	}

	exportVars := false

	tmpl, err := template.New(generatedMetadataTemplate).Option("missingkey=error").Funcs(template.FuncMap{
		"formatVariable": func(s string) (string, error) {
			formatted, err := formatVariable(s)
			if err != nil {
				return "", err
			}
			if exportVars {
				runes := []rune(formatted)
				runes[0] = unicode.ToUpper(runes[0])
				formatted = string(runes)
			}
			return formatted, nil
		},
		"convertMetricType": func(metricType string) (output string, err error) {
			switch metricType {
			case "gauge":
				return "datapoint.Gauge", nil
			case "counter":
				return "datapoint.Count", nil
			case "cumulative":
				return "datapoint.Counter", nil
			default:
				return "", fmt.Errorf("unknown metric type %s", metricType)
			}
		},
		"deref": func(p *string) string { return *p },
	}).ParseFiles(templateFile)

	if err != nil {
		return fmt.Errorf("parsing template %s failed: %s", generatedMetadataTemplate, err)
	}

	for i := range pkgs {
		pkg := &pkgs[i]
		if !force {
			if regenerate, err := shouldRegenerate(pkg, buildOutputPath(pkg)); err != nil {
				return err
			} else if !regenerate {
				continue
			}
		}

		writer := &bytes.Buffer{}
		groupMetricsMap := map[string][]string{}
		metrics := map[string]selfdescribe.MetricMetadata{}

		for _, mon := range pkg.Monitors {
			for metric, metricInfo := range mon.Metrics {
				if existingMetric, ok := metrics[metric]; ok {
					if existingMetric.Type != metricInfo.Type {
						return fmt.Errorf("existing metric %v does not have the same metric type as %v", existingMetric,
							metric)
					}
				} else {
					metrics[metric] = metricInfo
				}

				if metricInfo.Group != nil {
					metrics := []string{metric}
					if metricInfo.Alias != "" {
						metrics = append(metrics, metricInfo.Alias)
					}
					groupMetricsMap[*metricInfo.Group] = append(groupMetricsMap[*metricInfo.Group], metrics...)
				}
			}
		}

		// Pretty gross, resets variable that template function references above.
		exportVars = strings.TrimSpace(pkg.PackageDir) != ""

		// Go package name can be overridden but default to the directory name.
		var goPackage string
		switch {
		case exportVars:
			goPackage = pkg.PackageDir
		case pkg.GoPackage != nil:
			goPackage = *pkg.GoPackage
		default:
			goPackage = filepath.Base(pkg.PackagePath)
		}

		if err := tmpl.Execute(writer, map[string]interface{}{
			"metrics":         metrics,
			"monitors":        pkg.Monitors,
			"goPackage":       goPackage,
			"groupMetricsMap": groupMetricsMap,
		}); err != nil {
			return fmt.Errorf("failed executing template for %s: %s", pkg.Path, err)
		}

		formatted, err := format.Source(writer.Bytes())

		if err != nil {
			return fmt.Errorf("failed to format source: %s", err)
		}

		outputFile := buildOutputPath(pkg)

		if err := os.MkdirAll(path.Dir(outputFile), 0755); err != nil {
			return err
		}

		if err := ioutil.WriteFile(outputFile, formatted, 0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	force := flags.Bool("force", false, "set to force generate files")
	_ = flags.Parse(os.Args[1:])

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to determine filename")
	}

	if err := generate(path.Join(path.Dir(filename), generatedMetadataTemplate), *force); err != nil {
		log.Fatal(err)
	}
}
