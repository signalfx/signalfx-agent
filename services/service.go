package services

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/hopkinsth/go-ruler"
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
	Name string
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
func NewService(name string) *Service {
	return &Service{name}
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

// Matches if service instance satisfies rules
func (si *ServiceInstance) Matches(ruleset ServiceDiscoveryRuleset) (bool, error) {

	jsonRules, err := json.Marshal(ruleset.Rules)
	if err != nil {
		return false, err
	}

	engine, err := ruler.NewRulerWithJson(jsonRules)
	if err != nil {
		return false, err
	}

	sm := make(map[string]interface{})
	sm["ContainerID"] = si.Container.ID
	sm["ContainerName"] = si.Container.Names[0]
	sm["ContainerImage"] = si.Container.Image
	sm["ContainerPod"] = si.Container.Pod
	sm["ContainerCommand"] = si.Container.Command
	sm["ContainerState"] = si.Container.State

	for key, val := range si.Container.Labels {
		sm["ContainerLabel-"+key] = val
	}

	sm["NetworkIP"] = si.Port.IP
	sm["NetworkPublicPort"] = strconv.FormatUint(uint64(si.Port.PublicPort), 10)
	sm["NetworkPrivatePort"] = strconv.FormatUint(uint64(si.Port.PrivatePort), 10)
	sm["NetworkType"] = si.Port.Type

	for key, val := range si.Port.Labels {
		sm["NetworkLabel-"+key] = val
	}

	return engine.Test(sm), nil
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

// ServiceDiscoveryRule to use as criteria for service identification
type ServiceDiscoveryRule struct {
	Comparator string
	Path       string
	Value      interface{}
}

// NewServiceDiscoveryRule constructor
func NewServiceDiscoveryRule(comparator string, path string, value interface{}) *ServiceDiscoveryRule {
	return &ServiceDiscoveryRule{comparator, path, value}
}

// ServiceDiscoveryRuleset that names a set of service discovery rules
type ServiceDiscoveryRuleset struct {
	Name  string
	Rules []ServiceDiscoveryRule
}

// NewServiceDiscoveryRuleset constructor
func NewServiceDiscoveryRuleset(name string) *ServiceDiscoveryRuleset {
	return &ServiceDiscoveryRuleset{name, make([]ServiceDiscoveryRule, 0)}
}

// ServiceDiscoverySignatures with name
type ServiceDiscoverySignatures struct {
	Name       string
	Signatures []ServiceDiscoveryRuleset
}

// NewServiceDiscoverySignatures constructor
func NewServiceDiscoverySignatures(name string, signatures []ServiceDiscoveryRuleset) *ServiceDiscoverySignatures {
	return &ServiceDiscoverySignatures{name, signatures}
}

// LoadServiceSignatures reads discovery rules from file
func LoadServiceSignatures(servicesFile string) (ServiceDiscoverySignatures, error) {

	var signatures ServiceDiscoverySignatures
	jsonContent, err := ioutil.ReadFile(servicesFile)
	if err != nil {
		return signatures, err
	}

	if err := json.Unmarshal(jsonContent, &signatures); err != nil {
		return signatures, err
	}
	return signatures, nil
}
