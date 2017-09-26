package monitors

import (
	"strconv"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/config/types"
	"github.com/signalfx/neo-agent/core/services"
)

// This code is somewhat convoluted, but basically it creates two types of mock
// monitors, static and dynamic.  It handles doing basic tracking of whether
// the instances have been configured and how, so that we don't have to pry
// into the internals of the manager.

type Config struct {
	config.MonitorConfig
	MyVar            string
	MySlice          []string
	ServiceEndpoints []services.EndpointCore `yaml:"serviceEndpoints"`
}

type MockMonitor interface {
	GetConfig() *Config
	SetConfigHook(func(MockMonitor))
	AddShutdownHook(fn func())
}

type MockServiceMonitor interface {
	MockMonitor
	GetServices() map[services.ID]services.Endpoint
}

type _MockMonitor struct {
	Conf             *Config
	shutdownHooks    []func()
	configHook       func(MockMonitor)
	configHookCalled bool
}

type _MockServiceMonitor struct {
	_MockMonitor
	Services map[services.ID]services.Endpoint
}

var lastID = 0

func ensureID(conf *Config) {
	if string(conf.ID) == "" {
		conf.ID = types.MonitorID(strconv.Itoa(lastID))
		lastID++
	}
}

func (mb *_MockMonitor) Configure(conf *Config) bool {
	ensureID(conf)
	mb.Conf = conf
	if !mb.configHookCalled {
		mb.configHook(mb)
		mb.configHookCalled = true
	}
	return true
}

func (mb *_MockMonitor) GetConfig() *Config {
	return mb.Conf
}

func (mb *_MockMonitor) SetConfigHook(fn func(MockMonitor)) {
	mb.configHook = fn
}

func (mb *_MockMonitor) AddShutdownHook(fn func()) {
	mb.shutdownHooks = append(mb.shutdownHooks, fn)
}

func (mb *_MockMonitor) Shutdown() {
	for _, hook := range mb.shutdownHooks {
		hook()
	}
}

func (mb *_MockServiceMonitor) Configure(conf *Config) bool {
	ensureID(conf)
	mb.Conf = conf
	if !mb.configHookCalled {
		mb.configHook(mb)
		mb.configHookCalled = true
	}
	return true
}
func (mb *_MockServiceMonitor) AddService(service services.Endpoint) {
	if mb.Services == nil {
		mb.Services = make(map[services.ID]services.Endpoint)
	}
	mb.Services[service.ID()] = service
}

func (mb *_MockServiceMonitor) RemoveService(service services.Endpoint) {
	delete(mb.Services, service.ID())
}

func (mb *_MockServiceMonitor) GetServices() map[services.ID]services.Endpoint {
	return mb.Services
}

type Static1 struct{ _MockMonitor }
type Static2 struct{ _MockMonitor }
type Dynamic1 struct{ _MockServiceMonitor }
type Dynamic2 struct{ _MockServiceMonitor }

func RegisterFakeMonitors() func() []MockMonitor {
	instances := map[types.MonitorID]MockMonitor{}

	track := func(factory func() interface{}) func() interface{} {
		return func() interface{} {
			mon := factory().(MockMonitor)
			mon.SetConfigHook(func(mon MockMonitor) {
				instances[mon.GetConfig().ID] = mon
			})
			mon.AddShutdownHook(func() {
				delete(instances, mon.GetConfig().ID)
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
