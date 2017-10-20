package monitors

import (
	"time"

	"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/core/services"
	log "github.com/sirupsen/logrus"
)

var id = 0

func newService(imageName string, publicPort int) services.Endpoint {
	id++

	endpoint := services.NewEndpointCore(string(id), "", time.Now(), "test")
	endpoint.Port = uint16(publicPort)

	return &services.ContainerEndpoint{
		EndpointCore:  *endpoint,
		AltPort:       0,
		Container:     services.Container{Image: imageName},
		Orchestration: services.Orchestration{},
	}
}

var _ = Describe("Monitor Manager", func() {
	var manager *MonitorManager
	var getMonitors func() []MockMonitor

	BeforeEach(func() {
		DeregisterAll()

		getMonitors = RegisterFakeMonitors()

		manager = &MonitorManager{}
	})

	It("Starts up static monitors immediately", func() {
		log.SetLevel(log.DebugLevel)
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		spew.Dump(getMonitors())
		Expect(len(getMonitors())).To(Equal(1))
		mon := getMonitors()[0]
		Expect(mon.GetConfig().Type).To(Equal("static1"))
	})

	It("Shuts down static monitors when removed from config", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		Expect(len(getMonitors())).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "my-service"`,
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
				DiscoveryRule: `containerImage =~ "their-service"`,
			},
		})

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
	})

	It("Stops monitoring service if new monitor config no longer matches", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "their-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(0))
	})

	It("Monitors the same service on multiple monitors", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		manager.ServiceAdded(newService("my-service", 5000))

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
			config.MonitorConfig{
				Type:          "dynamic2",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))

		mons = findMonitorsByType(getMonitors(), "dynamic2")
		Expect(len(mons)).To(Equal(1))

		// Test restarting and making sure it still only monitors one service
		// each
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type:          "dynamic1",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
			config.MonitorConfig{
				Type:          "dynamic2",
				DiscoveryRule: `containerImage =~ "my-service"`,
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
		Expect(len(mons[0].(MockServiceMonitor).(MockServiceMonitor).GetServices())).To(Equal(1))

		mons = findMonitorsByType(getMonitors(), "dynamic2")
		Expect(len(mons)).To(Equal(1))
		Expect(len(mons[0].(MockServiceMonitor).(MockServiceMonitor).GetServices())).To(Equal(1))
	})

	It("Adds manually configured services to monitors", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type: "dynamic1",
				OtherConfig: map[string]interface{}{
					"serviceEndpoints": []interface{}{
						services.EndpointCore{
							Host: "myhost",
							Port: 5000,
						},
						services.EndpointCore{
							Host: "myhost2",
							Port: 5002,
						},
					},
				},
			},
		})

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
		Expect(len(mons[0].(MockServiceMonitor).GetServices())).To(Equal(2))
	})

	It("Removes manually configured services from monitors", func() {
		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type: "dynamic1",
				OtherConfig: map[string]interface{}{
					"serviceEndpoints": []interface{}{
						services.EndpointCore{
							Host: "myhost",
							Port: 5000,
						},
						services.EndpointCore{
							Host: "myhost2",
							Port: 5002,
						},
					},
				},
			},
		})

		mons := findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
		Expect(len(mons[0].(MockServiceMonitor).GetServices())).To(Equal(2))

		manager.Configure([]config.MonitorConfig{
			config.MonitorConfig{
				Type: "static1",
			},
			config.MonitorConfig{
				Type: "dynamic1",
				OtherConfig: map[string]interface{}{
					"serviceEndpoints": []interface{}{
						services.EndpointCore{
							MID:  "abcdef",
							Host: "myhost",
						},
					},
				},
			},
		})

		mons = findMonitorsByType(getMonitors(), "dynamic1")
		Expect(len(mons)).To(Equal(1))
		Expect(len(mons[0].(MockServiceMonitor).GetServices())).To(Equal(1))
		Expect(mons[0].(MockServiceMonitor).GetServices()["abcdef"]).To(Not(BeNil()))
	})

})
