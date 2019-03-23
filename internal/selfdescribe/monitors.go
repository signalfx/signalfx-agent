package selfdescribe

import (
	"go/doc"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

// Monitor metadata file.
const monitorMetadataFile = "metadata.yaml"

type metricMetadata struct {
	Name        string `json:"name"`
	Alias       string `json:"alias,omitempty"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Included    bool   `json:"included" default:"false"`
}

type propMetadata struct {
	Name        string `json:"name"`
	Dimension   string `json:"dimension"`
	Description string `json:"description"`
}

type monitorDocMetadata struct {
	MonitorType string           `yaml:"monitorType"`
	SendAll     bool             `yaml:"sendAll"`
	Dimensions  []dimMetadata    `yaml:"dimensions"`
	Metrics     []metricMetadata `yaml:"metrics"`
	Properties  []propMetadata   `yaml:"properties"`
	Doc         string           `yaml:"doc"`
}

type monitorMetadata struct {
	structMetadata
	AcceptsEndpoints bool             `json:"acceptsEndpoints"`
	SingleInstance   bool             `json:"singleInstance"`
	MonitorType      string           `json:"monitorType"`
	SendAll          bool             `json:"sendAll"`
	Dimensions       []dimMetadata    `json:"dimensions"`
	Metrics          []metricMetadata `json:"metrics"`
	Properties       []propMetadata   `json:"properties"`
}

func monitorsStructMetadata() []monitorMetadata {
	sms := []monitorMetadata{}
	// Set to track undocumented monitors
	monTypesSeen := make(map[string]bool)

	if err := filepath.Walk("internal/monitors", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || info.Name() != monitorMetadataFile {
			return nil
		}

		var monitorDocs []monitorDocMetadata

		if bytes, err := ioutil.ReadFile(path); err != nil {
			return errors.Errorf("Unable to read metadata file %s: %s", path, err)
		} else if err := yaml.UnmarshalStrict(bytes, &monitorDocs); err != nil {
			return err
		}

		for _, monitor := range monitorDocs {
			monType := monitor.MonitorType

			if _, ok := monitors.ConfigTemplates[monType]; !ok {
				log.Errorf("Found metadata for %s monitor in %s but it doesn't appear to be registered",
					monType, path)
				continue
			}
			t := reflect.TypeOf(monitors.ConfigTemplates[monType]).Elem()
			monTypesSeen[monType] = true

			checkSendAllLogic(monType, monitor.Metrics, monitor.SendAll)
			checkDuplicateMetrics(path, monitor.Metrics)
			checkMetricTypes(path, monitor.Metrics)

			mc, _ := t.FieldByName("MonitorConfig")
			mmd := monitorMetadata{
				structMetadata:   getStructMetadata(t),
				SendAll:          monitor.SendAll,
				MonitorType:      monType,
				Dimensions:       monitor.Dimensions,
				Metrics:          monitor.Metrics,
				Properties:       monitor.Properties,
				AcceptsEndpoints: mc.Tag.Get("acceptsEndpoints") == "true",
				SingleInstance:   mc.Tag.Get("singleInstance") == "true",
			}
			mmd.Doc = monitor.Doc
			mmd.Package = filepath.Dir(path)

			sms = append(sms, mmd)
		}

		return nil
	}); err != nil {
		log.Fatal(err)
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

func dimensionsFromNotes(allDocs []*doc.Package) []dimMetadata {
	var dm []dimMetadata
	for _, note := range notesFromDocs(allDocs, "DIMENSION") {
		dm = append(dm, dimMetadata{
			Name:        note.UID,
			Description: commentTextToParagraphs(note.Body),
		})
	}
	sort.Slice(dm, func(i, j int) bool {
		return dm[i].Name < dm[j].Name
	})
	return dm
}

func checkDuplicateMetrics(path string, metrics []metricMetadata) {
	seen := map[string]bool{}

	for i := range metrics {
		if seen[metrics[i].Name] {
			log.Errorf("duplicate metric '%s' found in %s", metrics[i].Name, path)
		}
		seen[metrics[i].Name] = true
	}
}

func checkMetricTypes(path string, metrics []metricMetadata) {
	for i := range metrics {
		t := metrics[i].Type
		if t != "gauge" && t != "counter" && t != "cumulative" {
			log.Errorf("Bad metric type '%s' for metric %s in %s", t, metrics[i].Name, path)
		}
	}
}

func checkSendAllLogic(monType string, metrics []metricMetadata, sendAll bool) {
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
