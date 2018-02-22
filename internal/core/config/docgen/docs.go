package docgen

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

type structMetadata struct {
	Name   string          `json:"name"`
	Doc    string          `json:"doc"`
	Fields []fieldMetadata `json:"fields"`
}

type monitorMetadata struct {
	structMetadata
	MonitorType      string `json:"monitorType"`
	AcceptsEndpoints bool   `json:"acceptsEndpoints"`
	SingleInstance   bool   `json:"singleInstance"`
}

type fieldMetadata struct {
	YAMLName string `json:"yamlName"`
	Doc      string `json:"doc"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	// Element is the metadata for the element type of a slice or the value
	// type of a map if they are structs.
	ElementStruct *structMetadata `json:"elementStruct,omitempty"`
}

var embeddedExclusions = map[string]bool{
	"MonitorConfig":  true,
	"ObserverConfig": true,
}

func packageDirOfType(t reflect.Type) string {
	return strings.TrimPrefix(t.PkgPath(), "github.com/signalfx/signalfx-agent/")
}

func getStructMetadata(typ reflect.Type) structMetadata {
	packageDir := packageDirOfType(typ)
	structName := typ.Name()
	if packageDir == "" || structName == "" {
		return structMetadata{}
	}

	fieldMD := []fieldMetadata{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		if f.Anonymous && !embeddedExclusions[f.Name] {
			nestedSM := getStructMetadata(f.Type)
			fieldMD = append(fieldMD, nestedSM.Fields...)
			continue
			// Embedded struct name and doc is irrelevant.
		}

		yamlName := getYAMLName(f)
		if yamlName == "" || yamlName == "-" {
			continue
		}

		fm := fieldMetadata{
			YAMLName: yamlName,
			Doc:      structFieldDocs(packageDir, structName)[f.Name],
			Default:  getDefault(f),
			Required: getRequired(f),
			Type:     indirectKind(f.Type).String(),
		}

		if indirectKind(f.Type) == reflect.Struct {
			smd := getStructMetadata(indirectType(f.Type))
			fm.ElementStruct = &smd
		} else if (f.Type.Kind() == reflect.Map || f.Type.Kind() == reflect.Slice) && indirectKind(f.Type.Elem()) == reflect.Struct {
			smd := getStructMetadata(indirectType(f.Type.Elem()))
			fm.ElementStruct = &smd
		}

		fieldMD = append(fieldMD, fm)
	}

	return structMetadata{
		Name:   structName,
		Doc:    structDoc(packageDir, structName),
		Fields: fieldMD,
	}
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
		}
		mmd.Doc = monitorDocFromPackageDoc(k, pkgDoc)

		sms = append(sms, mmd)
	}
	sort.Slice(sms, func(i, j int) bool {
		return sms[i].Name < sms[j].Name
	})
	return sms
}

func observersStructMetadata() map[string]structMetadata {
	sms := map[string]structMetadata{}
	for k := range observers.ConfigTemplates {
		t := reflect.TypeOf(observers.ConfigTemplates[k]).Elem()
		sms[k] = getStructMetadata(t)
	}
	return sms
}

// ConfigDocJSON returns a json encoded string of all of the documentation for
// the various config structures in the agent.  It is meant to be used as an
// intermediate form which serves as a data source for automatically generating
// docs about the agent.
func ConfigDocJSON() string {
	out, err := json.MarshalIndent(map[string]interface{}{
		"TopConfig":      getStructMetadata(reflect.TypeOf(config.Config{})),
		"MonitorConfig":  getStructMetadata(reflect.TypeOf(config.MonitorConfig{})),
		"ObserverConfig": getStructMetadata(reflect.TypeOf(config.ObserverConfig{})),
		"Monitors":       monitorsStructMetadata(),
		"Observers":      observersStructMetadata(),
	}, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(out)
}
