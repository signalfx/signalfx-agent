package services

import "time"

// OrchestrationType of service
type OrchestrationType int

// PortType An IP port type
type PortType string

// PortPreference public or private
type PortPreference int

// ServiceType A service/plugin type
type ServiceType string

const (
	// ApacheService Apache web server
	ApacheService ServiceType = "apache"
	// CassandraService Cassandra database
	CassandraService ServiceType = "cassandra"
	// ElasticSearchService ElasticSearch server
	ElasticSearchService ServiceType = "elasticsearch"
	// DockerService Docker container engine
	DockerService ServiceType = "docker"
	// KafkaService Kafka message broker
	KafkaService ServiceType = "kafka"
	// MemcachedService Memcached memory object store
	MemcachedService ServiceType = "memcached"
	// MongoDBService MongoDB database
	MongoDBService ServiceType = "mongodb"
	// MysqlService Mysql database
	MysqlService ServiceType = "mysql"
	// NginxService Nginx server
	NginxService ServiceType = "nginx"
	// RedisService Redis server
	RedisService ServiceType = "redis"
	// RabbitmqService Rabbitmq server
	RabbitmqService ServiceType = "rabbitmq"
	// SignalfxService SignalFx plugins
	SignalfxService ServiceType = "signalfx"
	// VarnishService Varnish cache
	VarnishService ServiceType = "varnish"
	// ZookeeperService Zookeeper server
	ZookeeperService ServiceType = "zookeeper"
)

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

// Service that can be discovered and monitored
type Service struct {
	Name string
	Type ServiceType
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
	ID       string
	Type     OrchestrationType
	Dims     map[string]string
	PortPref PortPreference
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
func NewService(name string, serviceType ServiceType) *Service {
	return &Service{name, serviceType}
}

// NewServicePort constructor
func NewServicePort(ip string, portType string, privatePort uint16, publicPort uint16) *ServicePort {
	return &ServicePort{ip, portType, privatePort, publicPort, make(map[string]string)}
}

// NewServiceOrchestration constructor
func NewServiceOrchestration(id string, orchType OrchestrationType, dims map[string]string, portPref PortPreference) *ServiceOrchestration {
	return &ServiceOrchestration{id, orchType, dims, portPref}
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
