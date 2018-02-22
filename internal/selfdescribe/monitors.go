package selfdescribe

import (
	"go/doc"
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

type dimMetadata struct {
	Name        string `json:"name"`
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
	for k := range monitors.ConfigTemplates {
		t := reflect.TypeOf(monitors.ConfigTemplates[k]).Elem()
		pkgDoc := packageDoc(packageDirOfType(t))

		mc, _ := t.FieldByName("MonitorConfig")
		mmd := monitorMetadata{
			structMetadata:   getStructMetadata(t),
			MonitorType:      k,
			AcceptsEndpoints: mc.Tag.Get("acceptsEndpoints") == "true",
			SingleInstance:   mc.Tag.Get("singleInstance") == "true",
			Dimensions:       dimensionsFromNotes(pkgDoc),
			Metrics:          metricsFromNotes(pkgDoc),
			Properties:       propertiesFromNotes(pkgDoc),
		}
		mmd.Doc = monitorDocFromPackageDoc(k, pkgDoc)

		sms = append(sms, mmd)
	}
	sort.Slice(sms, func(i, j int) bool {
		return sms[i].MonitorType < sms[j].MonitorType
	})
	return sms
}

func monitorDocFromPackageDoc(monitorType string, pkgDoc *doc.Package) string {
	for _, note := range pkgDoc.Notes["MONITOR"] {
		if note.UID == monitorType {
			return note.Body
		}
	}
	return ""
}

func dimensionsFromNotes(pkgDoc *doc.Package) []dimMetadata {
	var dm []dimMetadata
	for _, note := range pkgDoc.Notes["DIMENSION"] {
		dm = append(dm, dimMetadata{
			Name:        note.UID,
			Description: note.Body,
		})
	}
	return dm
}

func metricsFromNotes(pkgDoc *doc.Package) []metricMetadata {
	var mm []metricMetadata
	for _, note := range pkgDoc.Notes["GAUGE"] {
		mm = append(mm, metricMetadata{
			Type:        "gauge",
			Name:        note.UID,
			Description: note.Body,
		})
	}
	for _, note := range pkgDoc.Notes["TIMESTAMP"] {
		mm = append(mm, metricMetadata{
			Type:        "timestamp",
			Name:        note.UID,
			Description: note.Body,
		})
	}
	for _, note := range pkgDoc.Notes["COUNTER"] {
		mm = append(mm, metricMetadata{
			Type:        "counter",
			Name:        note.UID,
			Description: note.Body,
		})
	}
	for _, note := range pkgDoc.Notes["CUMULATIVE"] {
		mm = append(mm, metricMetadata{
			Type:        "cumulative counter",
			Name:        note.UID,
			Description: note.Body,
		})
	}
	return mm
}

func propertiesFromNotes(pkgDoc *doc.Package) []propMetadata {
	var pm []propMetadata
	for _, note := range pkgDoc.Notes["PROPERTY"] {
		parts := strings.Split(note.UID, ":")
		if len(parts) != 2 {
			log.Errorf("Property comment 'PROPERTY(%s): %s' in package %s should have form "+
				"'PROPERTY(propname:dimension_name): desc'", note.UID, note.Body, pkgDoc.Name)
		}

		pm = append(pm, propMetadata{
			Name:        parts[0],
			Dimension:   parts[1],
			Description: note.Body,
		})
	}
	return pm
}
