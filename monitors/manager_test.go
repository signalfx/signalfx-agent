package monitors

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/observers"
)

var id = 0

func newService(imageName string, publicPort int) *observers.ServiceInstance {
	id++
	return &observers.ServiceInstance{
		ID: observers.ServiceID(id),
		Container: &observers.Container{
			Image: imageName,
		},
		Port: &observers.Port{
			PublicPort: uint16(publicPort),
		},
	}
}

var _ = Describe("Monitor Manager", func() {
	var manager *MonitorManager
	var getMonitors func() []BaseMonitor

	BeforeEach(func() {
		DeregisterAll()

		getMonitors = RegisterFakeMonitors()

		manager = &MonitorManager{}
	})

	It("Starts up static monitors immediately", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		Expect(len(getMonitors())).To(Equal(1))
		mon := getMonitors()[0]
		Expect(mon.(*Static1).Conf.Type).To(Equal("static1"))
	})

	It("Shuts down static monitors when removed from config", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		Expect(len(getMonitors())).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		Expect(len(getMonitors())).To(Equal(0))
	})

	It("Starts up dynamic monitors upon service discovery", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		Expect(len(getMonitors())).To(Equal(1))

		manager.ServiceAdded(newService("my-service", 5000))

		Expect(len(getMonitors())).To(Equal(2))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
	})

	It("Shuts down dynamic monitors upon only service removed", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		service := newService("my-service", 5000)
		manager.ServiceAdded(service)

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		shutdownCalled := false
		mons[0].AddShutdownHook(func() {
			shutdownCalled = true
		})

		manager.ServiceRemoved(service)

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))
		Expect(shutdownCalled).To(Equal(true))
	})

	It("Shuts down dynamic monitor after multiple services removed", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		service := newService("my-service", 5000)
		service2 := newService("my-service", 5001)
		manager.ServiceAdded(service)
		manager.ServiceAdded(service2)

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		shutdownCalled := false
		mons[0].AddShutdownHook(func() {
			shutdownCalled = true
		})

		manager.ServiceRemoved(service)

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
		Expect(shutdownCalled).To(Equal(false))

		manager.ServiceRemoved(service2)

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))
		Expect(shutdownCalled).To(Equal(true))
	})

	It("Re-monitors service if monitor is removed temporarily", func() {
		goodConfig := []config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		}
		manager.Configure(goodConfig)

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
				OtherConfig:   map[string]interface{}{"invalid": true},
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))

		manager.Configure(goodConfig)

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
	})

	It("Starts monitoring previously discovered service if new monitor config matches", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "their-service"`,
			},
		})

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
	})

	It("Stops monitoring service if new monitor config no longer matches", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "my-service"`,
			},
		})

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `ContainerImage =~ "their-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))
	})
})
