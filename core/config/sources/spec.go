package sources

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/utils"
	yaml "gopkg.in/yaml.v2"
)

type dynamicValueSpec struct {
	From     *specURL `yaml:"#from"`
	Flatten  bool     `yaml:"flatten"`
	Optional bool     `yaml:"optional"`
}

type specURL struct {
	*url.URL
}

func (sp *specURL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	err := unmarshal(&str)
	if err != nil {
		return err
	}
	url, err := url.Parse(str)
	if err != nil {
		return err
	}
	sp.URL = url
	return nil
}

func (sp *specURL) SourceName() string {
	return utils.FirstNonEmpty(sp.URL.Scheme, "file")
}

func (sp *specURL) Path() string {
	return sp.URL.Path
}

// RawDynamicValueSpec is a string that should deserialize to a dynamic value
// spec (e.g. {"#from": "/path/to/value"}).
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
