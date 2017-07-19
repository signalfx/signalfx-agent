package monitors

import (
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
)

type Config struct {
	config.MonitorConfig
	MyVar   string
	MySlice []string
}

type MonBase struct {
	Conf          *Config
	shutdownHooks []func()
}

func (mb *MonBase) Configure(conf *Config) bool {
	print("Configure called ", conf.Type)
	mb.Conf = conf
	return true
}

func (mb *MonBase) GetConfig() *Config {
	return mb.Conf
}

func (mb *MonBase) AddShutdownHook(fn func()) {
	mb.shutdownHooks = append(mb.shutdownHooks, fn)
}

func (mb *MonBase) Shutdown() {
	for _, hook := range mb.shutdownHooks {
		hook()
	}
}

type DynamicBase struct {
	MonBase
	Services map[observers.ServiceID]*observers.ServiceInstance
}

func (db *DynamicBase) AddService(service *observers.ServiceInstance) {
	if db.Services == nil {
		db.Services = make(map[observers.ServiceID]*observers.ServiceInstance)
	}
	db.Services[service.ID] = service
}

func (db *DynamicBase) RemoveService(service *observers.ServiceInstance) {
	delete(db.Services, service.ID)
}

type BaseMonitor interface {
	AddShutdownHook(func())
	GetConfig() *Config
}

type Static1 struct{ MonBase }
type Static2 struct{ MonBase }
type Dynamic1 struct{ DynamicBase }
type Dynamic2 struct{ DynamicBase }

func RegisterFakeMonitors() func() []BaseMonitor {
	lastId := 0
	instances := map[int]interface{}{}

	track := func(factory func() interface{}) func() interface{} {
		return func() interface{} {
			mon := factory()
			id := lastId
			lastId++
			instances[id] = mon
			mon.(BaseMonitor).AddShutdownHook(func() {
				delete(instances, id)
			})

			return mon
		}
	}

	Register("static1", track(func() interface{} { return &Static1{} }), &Config{})
	Register("static2", track(func() interface{} { return &Static2{} }), &Config{})
	Register("dynamic1", track(func() interface{} { return &Dynamic1{} }), &Config{})
	Register("dynamic2", track(func() interface{} { return &Dynamic2{} }), &Config{})

	return func() []BaseMonitor {
		slice := []BaseMonitor{}
		for i := range instances {
			slice = append(slice, instances[i].(BaseMonitor))
		}
		return slice
	}
}

func findMonitorsByType(monitors []BaseMonitor, _type string) []BaseMonitor {
	mons := []BaseMonitor{}
	for i := range monitors {
		if monitors[i].GetConfig().Type == _type {
			mons = append(mons, monitors[i])
		}
	}
	return mons
}
