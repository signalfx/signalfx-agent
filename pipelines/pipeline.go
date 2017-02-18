package pipelines

import (
	"errors"

	"fmt"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
)

// Source is an interface that only produces service instances
type Source interface {
	Read() (services.ServiceInstances, error)
}

// Sink is an interface that only consumes service instances
type Sink interface {
	Write(services.ServiceInstances) error
}

// SourceSink is an interface that consumes and products service instances
type SourceSink interface {
	Map(services.ServiceInstances) (services.ServiceInstances, error)
}

// Pipeline is a series of sink/source plugins to execute in a specific order
type Pipeline struct {
	name    string
	plugins []plugins.IPlugin
}

// NewPipeline creates a Pipeline instance
func NewPipeline(name string, pluginNames []string, plugins []plugins.IPlugin) (*Pipeline, error) {
	pipeline := &Pipeline{name, nil}

	// TODO: Should probably have a map of name -> plugin instance.
	for _, pluginName := range pluginNames {
		found := false
		for _, pluginInstance := range plugins {
			if pluginName == pluginInstance.Name() {
				pipeline.plugins = append(pipeline.plugins, pluginInstance)
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("unable to find plugin instance for %s", pluginName)
		}
	}

	return pipeline, nil
}

// Execute runs steps sequentially in the pipeline (returns after all stages have been run once)
func (pipeline *Pipeline) Execute() error {
	var services services.ServiceInstances
	var err error

	for i, s := range pipeline.plugins {
		// Verify that the first entry in the pipeline isn't a a sink.
		if i == 0 {
			if _, ok := s.(Sink); ok {
				return errors.New("first entry in pipeline must be a source or a source/sink")
			}
		}

		// TODO - send cloned observer results to each sink/source?
		switch t := s.(type) {
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
