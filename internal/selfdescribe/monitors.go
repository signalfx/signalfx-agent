package selfdescribe

import (
	"go/doc"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/signalfx-agent/internal/monitors"
)

type metricMetadata struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type propMetadata struct {
	Name        string `json:"name"`
	Dimension   string `json:"dimension"`
	Description string `json:"description"`
}

type monitorMetadata struct {
	structMetadata
	MonitorType      string           `json:"monitorType"`
	AcceptsEndpoints bool             `json:"acceptsEndpoints"`
	SingleInstance   bool             `json:"singleInstance"`
	Dimensions       []dimMetadata    `json:"dimensions"`
	Metrics          []metricMetadata `json:"metrics"`
	Properties       []propMetadata   `json:"properties"`
}

func monitorsStructMetadata() []monitorMetadata {
	sms := []monitorMetadata{}
	// Set to track undocumented monitors
	monTypesSeen := make(map[string]bool)

	filepath.Walk("internal/monitors", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() || err != nil {
			return err
		}
		pkgDoc := packageDoc(path)
		for monType, monDoc := range monitorDocsInPackage(pkgDoc) {
			if _, ok := monitors.ConfigTemplates[monType]; !ok {
				log.Errorf("Found MONITOR doc for monitor type %s but it doesn't appear to be registered", monType)
				continue
			}
			t := reflect.TypeOf(monitors.ConfigTemplates[monType]).Elem()
			monTypesSeen[monType] = true

			allDocs := nestedPackageDocs(path)

			mc, _ := t.FieldByName("MonitorConfig")
			mmd := monitorMetadata{
				structMetadata:   getStructMetadata(t),
				MonitorType:      monType,
				AcceptsEndpoints: mc.Tag.Get("acceptsEndpoints") == "true",
				SingleInstance:   mc.Tag.Get("singleInstance") == "true",
				Dimensions:       dimensionsFromNotes(allDocs),
				Metrics:          metricsFromNotes(allDocs),
				Properties:       propertiesFromNotes(allDocs),
			}
			mmd.Doc = monDoc
			mmd.Package = path

			sms = append(sms, mmd)
		}
		return nil
	})
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

func monitorDocsInPackage(pkgDoc *doc.Package) map[string]string {
	out := make(map[string]string)
	for _, note := range pkgDoc.Notes["MONITOR"] {
		out[note.UID] = note.Body
	}
	return out
}

func dimensionsFromNotes(allDocs []*doc.Package) []dimMetadata {
	var dm []dimMetadata
	for _, note := range notesFromDocs(allDocs, "DIMENSION") {
		dm = append(dm, dimMetadata{
			Name:        note.UID,
			Description: commentTextToParagraphs(note.Body),
		})
	}
	return dm
}

func metricsFromNotes(allDocs []*doc.Package) []metricMetadata {
	var mm []metricMetadata
	for noteMarker, metaType := range map[string]string{
		"GAUGE":      "gauge",
		"TIMESTAMP":  "timestamp",
		"COUNTER":    "counter",
		"CUMULATIVE": "cumulative",
	} {
		for _, note := range notesFromDocs(allDocs, noteMarker) {
			mm = append(mm, metricMetadata{
				Type:        metaType,
				Name:        note.UID,
				Description: commentTextToParagraphs(note.Body),
			})
		}
	}
	return mm
}

func propertiesFromNotes(allDocs []*doc.Package) []propMetadata {
	var pm []propMetadata
	for _, note := range notesFromDocs(allDocs, "PROPERTY") {
		parts := strings.Split(note.UID, ":")
		if len(parts) != 2 {
			log.Errorf("Property comment 'PROPERTY(%s): %s' in package %s should have form "+
				"'PROPERTY(dimension_name:prop_name): desc'", note.UID, note.Body, allDocs[0].Name)
		}

		pm = append(pm, propMetadata{
			Name:        parts[1],
			Dimension:   parts[0],
			Description: commentTextToParagraphs(note.Body),
		})
	}
	return pm
}
