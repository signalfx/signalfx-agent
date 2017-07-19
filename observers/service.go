package observers

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// OrchestrationType of service
type OrchestrationType int

// ServiceID uniquely identifies a service instance
type ServiceID string

// PortType An IP port type
type PortType string

// PortPreference public or private
type PortPreference int

const (
	// UDP port type
	UDP PortType = "UDP"
	// TCP port type
	TCP PortType = "TCP"
	// PRIVATE Port preference
	PRIVATE PortPreference = 1 + iota
	// PUBLIC Port preference
	PUBLIC
)

const (
	// KUBERNETES orchestrator
	KUBERNETES OrchestrationType = 1 + iota
	// MESOS orchestrator
	MESOS
	// SWARM orchestrator
	SWARM
	// DOCKER orchestrator
	DOCKER
	// NONE orchestrator
	NONE
)

// Port network information
type Port struct {
	Name        string
	IP          string
	Type        PortType
	PrivatePort uint16
	PublicPort  uint16
	Labels      map[string]string
}

// NewPort constructor
func NewPort(name string, ip string, portType PortType, privatePort uint16, publicPort uint16) *Port {
	return &Port{name, ip, portType, privatePort, publicPort, make(map[string]string)}
}

func (p *Port) String() string {
	return fmt.Sprintf("%#v", p)
}

// Orchestration information
type Orchestration struct {
	ID       string
	Type     OrchestrationType
	Dims     map[string]string
	PortPref PortPreference
}

// NewOrchestration constructor
func NewOrchestration(id string, orchType OrchestrationType, dims map[string]string, portPref PortPreference) *Orchestration {
	return &Orchestration{id, orchType, dims, portPref}
}

func (o *Orchestration) String() string {
	return fmt.Sprintf("%#v", o)
}

// Container information
type Container struct {
	ID        string
	Names     []string
	Image     string
	Pod       string
	Command   string
	State     string
	Labels    map[string]string
	Namespace string
}

func (c *Container) PrimaryName() string {
	if len(c.Names) > 0 {
		return strings.TrimLeft(c.Names[0], "/")
	}
	return ""
}

func (c *Container) String() string {
	return fmt.Sprintf("%#v", c)
}

// NewContainer constructor
func NewContainer(id string, names []string, image string, pod string, command string, state string, labels map[string]string, namespace string) *Container {
	return &Container{id, names, image, pod, command, state, labels, namespace}
}

// ServiceInstance information for single instance of a discovered service
type ServiceInstance struct {
	ID            ServiceID
	Container     *Container
	Orchestration *Orchestration
	Port          *Port
	Discovered    time.Time
	Vars          map[string]interface{}
}

// NewInstance constructor
func NewServiceInstance(id string, container *Container, orchestration *Orchestration, port *Port, discovered time.Time) *ServiceInstance {
	return &ServiceInstance{ServiceID(id), container, orchestration, port, discovered, nil}
}

// Equivalent determine if rhs and lhs and roughly equal (ignores Discovered time)
func (lhs *ServiceInstance) Equivalent(rhs *ServiceInstance) bool {
	// Quick check before doing DeepEqual.
	if lhs.ID != rhs.ID {
		return false
	}

	rhsCopy := *rhs
	// Ignore discovered time.
	rhsCopy.Discovered = lhs.Discovered
	// Have to take address of rhsCopy so that it's comparing pointers to
	// pointers.
	return reflect.DeepEqual(lhs, &rhsCopy)
}

func (si *ServiceInstance) PreferredPort() uint16 {
	if si.Orchestration.PortPref == PRIVATE {
		return si.Port.PrivatePort
	} else {
		return si.Port.PublicPort
	}
}

func (si *ServiceInstance) Host() string {
	return si.Port.IP
}

func (si *ServiceInstance) String() string {
	return fmt.Sprintf("<Service [%s]: container: %s; orchestration: %s; port: %s; discovered: %s",
		si.ID, si.Container.String(), si.Orchestration.String(), si.Port.String(), si.Discovered)
}
