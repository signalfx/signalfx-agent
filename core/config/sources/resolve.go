package sources

import (
	"fmt"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type resolveFunc func(v RawDynamicValueSpec) ([]interface{}, string, bool, error)

// The resolver is what aggregates together multiple source caches and converts
// raw dynamic value specs (e.g. the {"#from": ...} values) to the actual
// value.
type resolver struct {
	sources map[string]*configSourceCacher
}

func newResolver(sources map[string]*configSourceCacher) *resolver {
	return &resolver{
		sources: sources,
	}
}

func (r *resolver) Resolve(raw RawDynamicValueSpec) ([]interface{}, string, bool, error) {
	spec, err := parseRawSpec(raw)
	if err != nil {
		return nil, "", false, err
	}

	sourceName := spec.From.SourceName()
	source, ok := r.sources[sourceName]
	if !ok {
		return nil, "", false, fmt.Errorf("Source '%s' is not configured", sourceName)
	}

	contentMap, err := source.Get(spec.From.Path(), spec.Optional)
	if err != nil {
		return nil, "", false, errors.WithMessage(err,
			"could not resolve path "+spec.From.String())
	}

	value, err := convertFileBytesToValues(contentMap)

	return value, spec.From.Path(), spec.Flatten, err
}

func convertFileBytesToValues(content map[string][]byte) ([]interface{}, error) {
	var out []interface{}
	for path := range content {
		if len(content[path]) == 0 {
			continue
		}

		var v interface{}
		err := yaml.Unmarshal(content[path], &v)
		if err != nil {
			return nil, errors.WithMessage(err, "deserialization error at path "+path)
		}

		out = append(out, v)
	}
	return out, nil
}
