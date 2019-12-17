package sources

import (
	"fmt"

	"github.com/pkg/errors"

	yaml "gopkg.in/yaml.v2"
)

type resolveFunc func(v RawDynamicValueSpec) ([]interface{}, string, *dynamicValueSpec, error)

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

func (r *resolver) Resolve(raw RawDynamicValueSpec) ([]interface{}, string, *dynamicValueSpec, error) {
	spec, err := parseRawSpec(raw)
	if err != nil {
		return nil, "", nil, err
	}

	sourceName := spec.From.SourceName()
	source, ok := r.sources[sourceName]
	if !ok {
		return nil, "", nil, fmt.Errorf("source '%s' is not configured", sourceName)
	}

	contentMap, err := source.Get(spec.From.Path(), spec.Optional)
	if err != nil {
		return nil, "", nil, errors.WithMessage(err,
			"could not resolve path "+spec.From.String())
	}

	var value []interface{}
	if len(contentMap) == 0 && spec.Default != nil {
		value = []interface{}{
			spec.Default,
		}
	} else {
		value, err = convertFileBytesToValues(contentMap, spec.Raw)
	}

	return value, spec.From.Path(), spec, err
}

func convertFileBytesToValues(content map[string][]byte, raw bool) ([]interface{}, error) {
	var out []interface{}
	for path := range content {
		if len(content[path]) == 0 {
			continue
		}

		var v interface{}
		if raw {
			v = string(content[path])
		} else {
			err := yaml.Unmarshal(content[path], &v)
			if err != nil {
				return nil, errors.WithMessage(err, "deserialization error at path "+path)
			}
		}

		out = append(out, v)
	}
	return out, nil
}
