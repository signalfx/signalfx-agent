package services

import (
	"fmt"
	"strings"
)

// Container information
type Container struct {
	ID      string            `yaml:"containerID"`
	Names   []string          `yaml:"containerNames"`
	Image   string            `yaml:"containerImage"`
	Command string            `yaml:"containerCommand"`
	State   string            `yaml:"containerState"`
	Labels  map[string]string `yaml:"containerLabels"`
	// K8s specific
	Pod       string `yaml:"pod"`
	Namespace string `yaml:"namespace"`
}

// PrimaryName is the first container name, with all slashes stripped from the
// beginning.
func (c *Container) PrimaryName() string {
	if len(c.Names) > 0 {
		return strings.TrimLeft(c.Names[0], "/")
	}
	return ""
}

func (c *Container) String() string {
	return fmt.Sprintf("%#v", c)
}
