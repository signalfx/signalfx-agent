package pipelines

import (
	"errors"

	"fmt"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
)

// Source is an interface that only produces service instances
type Source interface {
	Read() (services.Instances, error)
}

// Sink is an interface that only consumes service instances
type Sink interface {
	Write(services.Instances) error
}

// SourceSink is an interface that consumes and products service instances
type SourceSink interface {
	Map(services.Instances) (services.Instances, error)
}

// Pipeline is a series of sink/source plugins to execute in a specific order
type Pipeline struct {
	name    string
	plugins []*plugins.Plugin
}

// NewPipeline creates a Pipeline instance
func NewPipeline(name string, pluginNames []string, plugins map[string]*plugins.Plugin) (*Pipeline, error) {
	pipeline := &Pipeline{name, nil}

	for _, pluginName := range pluginNames {
		pi, ok := plugins[pluginName]
		if !ok {
			return nil, fmt.Errorf("unable to find plugin instance for %s", pluginName)
		}
		pipeline.plugins = append(pipeline.plugins, pi)
	}

	return pipeline, nil
}

// Execute runs steps sequentially in the pipeline (returns after all stages have been run once)
func (pipeline *Pipeline) Execute() error {
	var services services.Instances
	var err error

	for i, s := range pipeline.plugins {
		// Verify that the first entry in the pipeline isn't a a sink.
		if i == 0 {
			if _, ok := s.Instance.(Sink); ok {
				return errors.New("first entry in pipeline must be a source or a source/sink")
			}
		}

		// TODO - send cloned observer results to each sink/source?
		switch t := s.Instance.(type) {
		case Source:
			if services, err = t.Read(); err != nil {
				return err
			}
		case Sink:
			if err := t.Write(services); err != nil {
				return err
			}
		case SourceSink:
			if services, err = t.Map(services); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s does not support source/sink interface", s)
		}
	}

	return nil
}
