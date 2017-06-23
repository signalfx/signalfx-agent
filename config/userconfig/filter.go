package userconfig

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// Filter - specifies filters to exclude containers from docker plugin and cadvisor
type Filter struct {
	DockerContainerNames     []string `yaml:"dockerContainerNames,omitempty"`
	Images                   []string `yaml:"images,omitempty,omitempty"`
	KubernetesContainerNames []string `yaml:"kubernetesContainerNames,omitempty"`
	KubernetesPodNames       []string `yaml:"kubernetesPodNames,omitempty"`
	KubernetesNamespaces     []string `yaml:"kubernetesNamespaces,omitempty"`
	Labels                   []*Label `yaml:"labels,omitempty"`
}

// LoadYaml - load yaml file
func (f *Filter) LoadYAML(path string) error {
	var err error
	var file []byte
	// Load the yaml file
	if file, err = ioutil.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(file, &f); err != nil {
			return err
		}
	}
	return err
}

// ParseConfig -
func (f *Filter) Parse(store map[string]interface{}) error {
	// assign image filters
	if len(f.Images) != 0 {
		store["excludedImages"] = f.Images
	}
	// assign docker container name filter
	if len(f.DockerContainerNames) != 0 {
		store["excludedNames"] = f.DockerContainerNames
	}
	// if there are labels add them
	if labels := f.GetLabels(); len(labels) > 0 {
		// append the lables filter
		store["excludedLabels"] = labels
	}

	return nil
}

// GetLabels -
func (f *Filter) GetLabels() []*Label {
	var labels = []*Label{}

	for _, label := range f.Labels {
		labels = append(labels, label)
	}

	// configure the label filters
	if len(f.Labels) != 0 || len(f.KubernetesNamespaces) != 0 || len(f.KubernetesContainerNames) != 0 || len(f.KubernetesPodNames) != 0 {
		// assign namespaces to labels because k8s namespace is actually a label
		for _, namespace := range f.KubernetesNamespaces {
			labels = append(labels, &Label{Key: "^io.kubernetes.pod.namespace$", Value: namespace})
		}
		// assign k8s container name to labels because k8s container name is actually a label
		for _, containerName := range f.KubernetesContainerNames {
			labels = append(labels, &Label{Key: "^io.kubernetes.container.name$", Value: containerName})
		}

		// assign k8s pod name to labels because k8s podname is actually a label
		for _, podName := range f.KubernetesPodNames {
			labels = append(labels, &Label{Key: "^io.kubernetes.pod.name$", Value: podName})
		}
	}
	return labels
}
