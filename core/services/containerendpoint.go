package services

import (
	"github.com/signalfx/neo-agent/utils"
)

// ContainerEndpoint contains information for single network endpoint of a
// discovered containerized service.  A single real-world service could have
// multiple distinct instances if it exposes multiple ports or is discovered by
// more than one observer.
type ContainerEndpoint struct {
	EndpointCore `yaml:",inline"`
	// altPort is used for services that are accessed through some kind of
	// NAT redirection as Docker does.  This could be either the public port
	// or the private one.
	AltPort       uint16            `yaml:"alternatePort"`
	Container     Container         `yaml:",inline"`
	Orchestration Orchestration     `yaml:",inline"`
	PortLabels    map[string]string `yaml:"portLabels"`
}

// PublicPort is the port that the endpoint is accessed on externally.  It may
// be different from the PrivatePort.
func (ce *ContainerEndpoint) PublicPort() uint16 {
	if ce.Orchestration.PortPref == PUBLIC {
		return ce.Port
	}
	return ce.AltPort
}

// PrivatePort is the port that the service is configured to listen on
func (ce *ContainerEndpoint) PrivatePort() uint16 {
	if ce.Orchestration.PortPref == PRIVATE {
		return ce.Port
	}
	return ce.AltPort
}

// DerivedFields returns aliased and computed fields for this endpoint
func (ce *ContainerEndpoint) DerivedFields() map[string]interface{} {
	return map[string]interface{}{
		"publicPort":    ce.PublicPort(),
		"privatePort":   ce.PrivatePort(),
		"containerName": ce.Container.PrimaryName(),
	}
}

// Dimensions returns the dimensions associated with this endpoint
func (ce *ContainerEndpoint) Dimensions() map[string]string {
	return utils.MergeStringMaps(ce.EndpointCore.Dimensions(), ce.Orchestration.Dimensions, map[string]string{
		"container_name":           ce.Container.PrimaryName(),
		"container_image":          ce.Container.Image,
		"kubernetes_pod_name":      ce.Container.Pod,
		"kubernetes_pod_namespace": ce.Container.Namespace,
		// This is essential as it is the only unique dim for a pod.
		"kubernetes_pod_uid": ce.Container.PodUID,
	})
}
