package sources

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/utils"
	yaml "gopkg.in/yaml.v2"
)

type dynamicValueSpec struct {
	From     *fromPath `yaml:"#from"`
	Flatten  bool      `yaml:"flatten"`
	Optional bool      `yaml:"optional"`
	Raw      bool      `yaml:"raw"`
}

type fromPath struct {
	sourceName string
	path       string
}

func (sp *fromPath) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	err := unmarshal(&str)
	if err != nil {
		return err
	}
	if len(str) == 0 {
		return errors.New("#from value is empty")
	}

	parts := strings.SplitN(str, ":", 2)

	if len(parts) == 1 {
		sp.path = parts[0]
	} else {
		sp.sourceName = parts[0]
		sp.path = parts[1]
	}

	return nil
}

func (sp *fromPath) SourceName() string {
	return utils.FirstNonEmpty(sp.sourceName, "file")
}

func (sp *fromPath) Path() string {
	return sp.path
}

func (sp *fromPath) String() string {
	return sp.SourceName() + ":" + sp.Path()
}

// RawDynamicValueSpec is a string that should deserialize to a dynamic value
// path (e.g. {"#from": "/path/to/value"}).
type RawDynamicValueSpec interface{}

func parseRawSpec(r RawDynamicValueSpec) (*dynamicValueSpec, error) {
	text, err := yaml.Marshal(r)
	if err != nil {
		return nil, err
	}

	var dvs dynamicValueSpec
	err = yaml.UnmarshalStrict(text, &dvs)

	if dvs.From == nil {
		// We should never get here for any given user input if the calling
		// code is doing its job.
		return nil, errors.New("#from field is missing")
	}

	return &dvs, err
}
