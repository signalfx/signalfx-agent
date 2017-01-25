package services

import (
	"time"
)

// OrchestrationType of service
type OrchestrationType int

const (
	// KUBERNETES orchestrator
	KUBERNETES OrchestrationType = 1 + iota
	// MESOS orchestrator
	MESOS
	// SWARM orchestrator
	SWARM
	// NONE orchestrator
	NONE
)

// Service that can be discovered and monitored
type Service struct {
	ID        string
	Name      string
	Discovery *ServiceDiscovery
}

// ServiceDiscovery rules
type ServiceDiscovery struct {
	File   string
	Loaded time.Time
	Rules  string
}

// ServicePort network information
type ServicePort struct {
	IP          string
	Type        string
	PrivatePort uint16
	PublicPort  uint16
	Labels      map[string]string
}

// ServiceOrchestration information
type ServiceOrchestration struct {
	ID        string
	Name      string
	Type      OrchestrationType
	AgentID   string
	AgentName string
}

// ServiceContainer information
type ServiceContainer struct {
	ID      string
	Names   []string
	Image   string
	Pod     string
	Command string
	State   string
	Labels  map[string]string
}

// ServiceInstance information for single instance of a discovered service
type ServiceInstance struct {
	ID            string
	Service       *Service
	Container     *ServiceContainer
	Orchestration *ServiceOrchestration
	Port          *ServicePort
	Discovered    time.Time
}

// NewService constructor
func NewService(id string, name string, file string) *Service {
	return &Service{id, name, NewServiceDiscovery(file)}
}

// NewServiceDiscovery constructor
func NewServiceDiscovery(file string) *ServiceDiscovery {
	// TODO - load rules from file
	rules := "{'id': 'match', 'port': 7099}"
	return &ServiceDiscovery{file, time.Now(), rules}
}

// NewServicePort constructor
func NewServicePort(ip string, portType string, privatePort uint16, publicPort uint16) *ServicePort {
	return &ServicePort{ip, portType, privatePort, publicPort, make(map[string]string)}
}

// NewServiceOrchestration constructor
func NewServiceOrchestration(id string, name string, orchType OrchestrationType, agentID string, agentName string) *ServiceOrchestration {
	return &ServiceOrchestration{id, name, orchType, agentID, agentName}
}

// NewServiceContainer constructor
func NewServiceContainer(id string, names []string, image string, pod string, command string, state string, labels map[string]string) *ServiceContainer {
	return &ServiceContainer{id, names, image, pod, command, state, labels}
}

// NewServiceInstance constructor
func NewServiceInstance(id string, service *Service, container *ServiceContainer, orchestration *ServiceOrchestration, port *ServicePort, discovered time.Time) *ServiceInstance {
	return &ServiceInstance{id, service, container, orchestration, port, discovered}
}

// ServiceInstances type containing sorted set of services
type ServiceInstances []ServiceInstance

// Len for serviceinstances sort
func (svcs ServiceInstances) Len() int {
	return len(svcs)
}

// Swap for serviceinstances sort
func (svcs ServiceInstances) Swap(i, j int) {
	svcs[i], svcs[j] = svcs[j], svcs[i]
}

// Less for serviceinstances sort
func (svcs ServiceInstances) Less(i, j int) bool {
	return svcs[i].ID < svcs[j].ID
}
