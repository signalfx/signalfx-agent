package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
)

type serviceEndpoint struct {
	services.EndpointCore `yaml:",inline"`
	ServiceURL            *string `yaml:"serviceURL"`
}

type Config struct {
	config.MonitorConfig
	MyVar            string
	MySlice          []string
	ServiceEndpoints []serviceEndpoint `yaml:"serviceEndpoints"`
}

type MockMonitor interface {
	GetConfig() *Config
	AddShutdownHook(fn func())
	GetServices() map[services.ID]services.Endpoint
}

type _MockMonitor struct {
	Conf          *Config
	shutdownHooks []func()
	Services      map[services.ID]services.Endpoint
}

func (mb *_MockMonitor) Configure(conf *Config) bool {
	print("Configure called ", conf.Type)
	mb.Conf = conf
	return true
}

func (mb *_MockMonitor) GetConfig() *Config {
	return mb.Conf
}

func (mb *_MockMonitor) AddShutdownHook(fn func()) {
	mb.shutdownHooks = append(mb.shutdownHooks, fn)
}

func (mb *_MockMonitor) Shutdown() {
	for _, hook := range mb.shutdownHooks {
		hook()
	}
}

func (mb *_MockMonitor) AddService(service services.Endpoint) {
	if mb.Services == nil {
		mb.Services = make(map[services.ID]services.Endpoint)
	}
	mb.Services[service.ID()] = service
}

func (mb *_MockMonitor) RemoveService(service services.Endpoint) {
	delete(mb.Services, service.ID())
}

func (mb *_MockMonitor) GetServices() map[services.ID]services.Endpoint {
	return mb.Services
}

type Static1 struct{ _MockMonitor }
type Static2 struct{ _MockMonitor }
type Dynamic1 struct{ _MockMonitor }
type Dynamic2 struct{ _MockMonitor }

func RegisterFakeMonitors() func() []MockMonitor {
	lastID := 0
	instances := map[int]MockMonitor{}

	track := func(factory func() interface{}) func() interface{} {
		return func() interface{} {
			mon := factory()
			id := lastID
			lastID++
			instances[id] = mon.(MockMonitor)
			mon.(MockMonitor).AddShutdownHook(func() {
				delete(instances, id)
			})

			return mon
		}
	}

	Register("static1", track(func() interface{} { return &Static1{} }), &Config{})
	Register("static2", track(func() interface{} { return &Static2{} }), &Config{})
	Register("dynamic1", track(func() interface{} { return &Dynamic1{} }), &Config{})
	Register("dynamic2", track(func() interface{} { return &Dynamic2{} }), &Config{})

	return func() []MockMonitor {
		slice := []MockMonitor{}
		for i := range instances {
			slice = append(slice, instances[i].(MockMonitor))
		}
		return slice
	}
}

func findMonitorsByType(monitors []MockMonitor, _type string) []MockMonitor {
	mons := []MockMonitor{}
	for i := range monitors {
		// Must check for nil since monitors can be created but not
		// successfully configured due to failed validation
		if monitors[i].GetConfig() != nil && monitors[i].GetConfig().Type == _type {
			mons = append(mons, monitors[i])
		}
	}
	return mons
}
