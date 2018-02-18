package docgen

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/observers"
)

type StructMetadata struct {
	Name   string          `json:"name"`
	Doc    string          `json:"doc"`
	Fields []FieldMetadata `json:"fields"`
}

type MonitorMetadata struct {
	StructMetadata
	AcceptsEndpoints bool `json:"acceptsEndpoints"`
	SingleInstance   bool `json:"singleInstance"`
}

type FieldMetadata struct {
	YAMLName string `json:"yamlName"`
	Doc      string `json:"doc"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	// Element is the metadata for the element type of a slice or the value
	// type of a map if they are structs.
	ElementStruct *StructMetadata `json:"elementStruct,omitempty"`
}

func getStructMetadata(typ reflect.Type) StructMetadata {
	pkg := typ.PkgPath()
	packageDir := strings.TrimPrefix(pkg, "github.com/signalfx/signalfx-agent/")
	structName := typ.Name()
	if packageDir == "" || structName == "" {
		return StructMetadata{}
	}

	fieldMD := []FieldMetadata{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		yamlName := getYAMLName(f)
		if yamlName == "" || yamlName == "-" {
			continue
		}

		if f.Anonymous {
			nestedSM := getStructMetadata(f.Type)
			fieldMD = append(fieldMD, nestedSM.Fields...)
			continue
			// Embedded struct name and doc is irrelevant.
		}

		fm := FieldMetadata{
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

	return StructMetadata{
		Name:   structName,
		Doc:    structDoc(packageDir, structName),
		Fields: fieldMD,
	}
}

func monitorsStructMetadata() map[string]MonitorMetadata {
	sms := map[string]MonitorMetadata{}
	for k := range monitors.ConfigTemplates {
		t := reflect.TypeOf(monitors.ConfigTemplates[k]).Elem()
		mc, _ := t.FieldByName("MonitorConfig")
		mmd := MonitorMetadata{
			StructMetadata:   getStructMetadata(t),
			AcceptsEndpoints: mc.Tag.Get("acceptsEndpoints") == "true",
			SingleInstance:   mc.Tag.Get("singleInstance") == "true",
		}

		sms[k] = mmd
	}
	return sms
}

func observersStructMetadata() map[string]StructMetadata {
	sms := map[string]StructMetadata{}
	for k := range observers.ConfigTemplates {
		sms[k] = getStructMetadata(reflect.TypeOf(observers.ConfigTemplates[k]).Elem())
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
