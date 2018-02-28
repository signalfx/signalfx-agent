package selfdescribe

import (
	"go/doc"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/observers"
	log "github.com/sirupsen/logrus"
)

type observerMetadata struct {
	structMetadata
	ObserverType      string        `json:"observerType"`
	Dimensions        []dimMetadata `json:"dimensions"`
	EndpointVariables []endpointVar `json:"endpointVariables"`
}

type endpointVar struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	ElementKind string `json:"elementKind"`
	Description string `json:"description"`
}

func observersStructMetadata() []observerMetadata {
	sms := []observerMetadata{}
	// Set to track undocumented observers
	obsTypesSeen := make(map[string]bool)

	filepath.Walk("internal/observers", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() || err != nil {
			return err
		}
		pkgDoc := packageDoc(path)
		if pkgDoc == nil {
			return nil
		}
		for obsType, obsDoc := range observerDocsInPackage(pkgDoc) {
			if _, ok := observers.ConfigTemplates[obsType]; !ok {
				log.Errorf("Found OBSERVER doc for observer type %s but it doesn't appear to be registered", obsType)
				continue
			}
			t := reflect.TypeOf(observers.ConfigTemplates[obsType]).Elem()
			obsTypesSeen[obsType] = true

			allDocs := nestedPackageDocs(path)

			mmd := observerMetadata{
				structMetadata:    getStructMetadata(t),
				ObserverType:      obsType,
				Dimensions:        dimensionsFromNotes(allDocs),
				EndpointVariables: endpointVariables(allDocs),
			}
			mmd.Doc = obsDoc
			mmd.Package = path

			sms = append(sms, mmd)
		}
		return nil
	})
	sort.Slice(sms, func(i, j int) bool {
		return sms[i].ObserverType < sms[j].ObserverType
	})

	for k := range observers.ConfigTemplates {
		if !obsTypesSeen[k] {
			log.Warnf("Observer Type %s is registered but does not appear to have documentation", k)
		}
	}

	return sms
}

func observerDocsInPackage(pkgDoc *doc.Package) map[string]string {
	out := make(map[string]string)
	for _, note := range pkgDoc.Notes["OBSERVER"] {
		out[note.UID] = note.Body
	}
	return out
}

func endpointVariables(obsDocs []*doc.Package) []endpointVar {
	servicesDocs := nestedPackageDocs("internal/core/services")
	obsEndpointTypes := notesFromDocs(obsDocs, "ENDPOINT_TYPE")

	var eType reflect.Type
	var includeContainerVars bool
	if len(obsEndpointTypes) > 0 && obsEndpointTypes[0].UID == "ContainerEndpoint" {
		eType = reflect.TypeOf(services.ContainerEndpoint{})
		includeContainerVars = true
	} else {
		eType = reflect.TypeOf(services.EndpointCore{})
	}
	sm := getStructMetadata(eType)

	return append(
		endpointVariablesFromNotes(append(obsDocs, servicesDocs...), includeContainerVars),
		endpointVarsFromStructMetadataFields(sm.Fields)...)
}

func endpointVarsFromStructMetadataFields(fields []fieldMetadata) []endpointVar {
	var endpointVars []endpointVar
	for _, fm := range fields {
		if fm.ElementStruct != nil {
			endpointVars = append(endpointVars, endpointVarsFromStructMetadataFields(fm.ElementStruct.Fields)...)
			continue
		}

		endpointVars = append(endpointVars, endpointVar{
			Name:        fm.YAMLName,
			Type:        fm.Type,
			ElementKind: fm.ElementKind,
			Description: fm.Doc,
		})
	}
	return endpointVars
}

func endpointVariablesFromNotes(allDocs []*doc.Package, includeContainerVars bool) []endpointVar {
	var pm []endpointVar
	for _, note := range notesFromDocs(allDocs, "ENDPOINT_VAR") {
		pm = append(pm, endpointVar{
			Name:        note.UID,
			Type:        "string",
			Description: commentTextToParagraphs(note.Body),
		})
	}

	// This is pretty hacky but is about the cleanest way to distinguish
	// container derived variables from non-container vars so that docs aren't
	// misleading.
	if includeContainerVars {
		for _, note := range notesFromDocs(allDocs, "CONTAINER_ENDPOINT_VAR") {
			pm = append(pm, endpointVar{
				Name:        note.UID,
				Type:        "string",
				Description: commentTextToParagraphs(note.Body),
			})
		}
	}
	return pm
}
