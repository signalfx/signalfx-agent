package selfdescribe

import (
	"go/doc"
	"reflect"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

type monitorDoc struct {
	monitors.MonitorMetadata
	Config           structMetadata `json:"config"`
	AcceptsEndpoints bool           `json:"acceptsEndpoints"`
	SingleInstance   bool           `json:"singleInstance"`
}

func monitorsStructMetadata() []monitorDoc {
	sms := []monitorDoc{}
	// Set to track undocumented monitors
	monTypesSeen := map[string]bool{}

	if packages, err := monitors.CollectMetadata("internal/monitors"); err != nil {
		log.Fatal(err)
	} else {
		for _, pkg := range packages {
			for _, monitor := range pkg.Monitors {
				monType := monitor.MonitorType

				if _, ok := monitors.ConfigTemplates[monType]; !ok {
					log.Errorf("Found metadata for %s monitor in %s but it doesn't appear to be registered",
						monType, pkg.Path)
					continue
				}
				t := reflect.TypeOf(monitors.ConfigTemplates[monType]).Elem()
				monTypesSeen[monType] = true

				checkSendAllLogic(monType, monitor.Metrics, monitor.SendAll)
				checkDuplicateMetrics(pkg.Path, monitor.Metrics)
				checkMetricTypes(pkg.Path, monitor.Metrics)

				mc, _ := t.FieldByName("MonitorConfig")
				mmd := monitorDoc{
					Config: getStructMetadata(t),
					MonitorMetadata: monitors.MonitorMetadata{
						SendAll:     monitor.SendAll,
						MonitorType: monType,
						Dimensions:  monitor.Dimensions,
						Groups:      monitor.Groups,
						Metrics:     monitor.Metrics,
						Properties:  monitor.Properties,
						Doc:         monitor.Doc,
					},
					AcceptsEndpoints: mc.Tag.Get("acceptsEndpoints") == strconv.FormatBool(true),
					SingleInstance:   mc.Tag.Get("singleInstance") == strconv.FormatBool(true),
				}
				mmd.Config.Package = pkg.PackagePath

				sms = append(sms, mmd)
			}
		}
	}

	sort.Slice(sms, func(i, j int) bool {
		return sms[i].MonitorType < sms[j].MonitorType
	})

	for k := range monitors.ConfigTemplates {
		if !monTypesSeen[k] {
			log.Warnf("Monitor Type %s is registered but does not appear to have documentation", k)
		}
	}

	return sms
}

func dimensionsFromNotes(allDocs []*doc.Package) []monitors.DimMetadata {
	var dm []monitors.DimMetadata
	for _, note := range notesFromDocs(allDocs, "DIMENSION") {
		dm = append(dm, monitors.DimMetadata{
			Name:        note.UID,
			Description: commentTextToParagraphs(note.Body),
		})
	}
	sort.Slice(dm, func(i, j int) bool {
		return dm[i].Name < dm[j].Name
	})
	return dm
}

func checkDuplicateMetrics(path string, metrics []monitors.MetricMetadata) {
	seen := map[string]bool{}

	for i := range metrics {
		if seen[metrics[i].Name] {
			log.Errorf("duplicate metric '%s' found in %s", metrics[i].Name, path)
		}
		seen[metrics[i].Name] = true
	}
}

func checkMetricTypes(path string, metrics []monitors.MetricMetadata) {
	for i := range metrics {
		t := metrics[i].Type
		if t != "gauge" && t != "counter" && t != "cumulative" {
			log.Errorf("Bad metric type '%s' for metric %s in %s", t, metrics[i].Name, path)
		}
	}
}

func checkSendAllLogic(monType string, metrics []monitors.MetricMetadata, sendAll bool) {
	if len(metrics) == 0 {
		return
	}

	hasIncluded := false
	for i := range metrics {
		hasIncluded = hasIncluded || metrics[i].Included
	}
	if hasIncluded && sendAll {
		log.Warnf("sendAll was specified on monitor type '%s' but some metrics were also marked as 'included'", monType)
	} else if !hasIncluded && !sendAll {
		log.Warnf("sendAll was not specified on monitor type '%s' and no metrics are marked as 'included'", monType)
	}
}
