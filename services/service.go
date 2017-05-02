package services

import "time"
import "reflect"

// OrchestrationType of service
type OrchestrationType int

// PortType An IP port type
type PortType string

// PortPreference public or private
type PortPreference int

// ServiceType A service/plugin type
type ServiceType string

const (
	// ActiveMQService messaging service
	ActiveMQService ServiceType = "activemq"
	// ApacheService Apache web server
	ApacheService ServiceType = "apache"
	// CassandraService Cassandra database
	CassandraService ServiceType = "cassandra"
	// ElasticSearchService ElasticSearch server
	ElasticSearchService ServiceType = "elasticsearch"
	// GenericJMXService custom jmx service
	GenericJMXService ServiceType = "genericjmx"
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
	// VarnishService Varnish cache
	VarnishService ServiceType = "varnish"
	// ZookeeperService Zookeeper server
	ZookeeperService ServiceType = "zookeeper"
	// UnknownService Unknown service
	UnknownService ServiceType = ""
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
	Name   string
	Type   ServiceType
	Plugin string
}

// Port network information
type Port struct {
	Name        string
	IP          string
	Type        PortType
	PrivatePort uint16
	PublicPort  uint16
	Labels      map[string]string
}

// Orchestration information
type Orchestration struct {
	ID       string
	Type     OrchestrationType
	Dims     map[string]string
	PortPref PortPreference
}

// Container information
type Container struct {
	ID      string
	Names   []string
	Image   string
	Pod     string
	Command string
	State   string
	Labels  map[string]string
}

// Instance information for single instance of a discovered service
type Instance struct {
	ID            string
	Service       *Service
	Container     *Container
	Orchestration *Orchestration
	Port          *Port
	Discovered    time.Time
	Vars          map[string]interface{}
	Template      string
}

// NewService constructor
// name should be unique enough for using as an id (host, instance, etc.)
func NewService(name string, serviceType ServiceType, plugin string) *Service {
	return &Service{name, serviceType, plugin}
}

// NewPort constructor
func NewPort(name string, ip string, portType PortType, privatePort uint16, publicPort uint16) *Port {
	return &Port{name, ip, portType, privatePort, publicPort, make(map[string]string)}
}

// NewOrchestration constructor
func NewOrchestration(id string, orchType OrchestrationType, dims map[string]string, portPref PortPreference) *Orchestration {
	return &Orchestration{id, orchType, dims, portPref}
}

// NewContainer constructor
func NewContainer(id string, names []string, image string, pod string, command string, state string, labels map[string]string) *Container {
	return &Container{id, names, image, pod, command, state, labels}
}

// NewInstance constructor
func NewInstance(id string, service *Service, container *Container, orchestration *Orchestration, port *Port, discovered time.Time) *Instance {
	return &Instance{id, service, container, orchestration, port, discovered, nil, ""}
}

// Equivalent determine if rhs and lhs and roughly equal (ignores Discovered time)
func (lhs *Instance) Equivalent(rhs *Instance) bool {
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

// Instances type containing sorted set of services
type Instances []Instance

// Len for serviceinstances sort
func (svcs Instances) Len() int {
	return len(svcs)
}

// Swap for serviceinstances sort
func (svcs Instances) Swap(i, j int) {
	svcs[i], svcs[j] = svcs[j], svcs[i]
}

// Less for serviceinstances sort
func (svcs Instances) Less(i, j int) bool {
	return svcs[i].ID < svcs[j].ID
}
