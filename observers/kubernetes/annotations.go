package kubernetes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/utils/k8sutil"
	log "github.com/sirupsen/logrus"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

var annotationConfigRegexp = regexp.MustCompile(
	`^agent.signalfx.com/` +
		`(?P<type>monitorType|config|configFromEnv|configFromSecret)` +
		`.(?P<port>[\w-]+)` +
		`(?:.(?P<config_key>\w+))?$`)

// AnnotationConfig is a generic struct that can describe any of the annotation
// config values we support.
type AnnotationConfig struct {
	AnnotationKey string
	// The type of annotation
	Type string
	// Either the port number or name must be specified
	Port     int32
	PortName string
	// The config key that this will result in when configuring a monitor
	ConfigKey string
	Value     string
}

// AnnotationConfigs is a slice of AnnotationConfig with some helper methods
// for filtering.
type AnnotationConfigs []*AnnotationConfig

// FilterByPortOrPortName returns all AnnotationConfig instances that match
// either the port number or port name.
func (ac AnnotationConfigs) FilterByPortOrPortName(port int32, portName string) (out AnnotationConfigs) {
	for i := range ac {
		if ac[i].Port == port || (portName != "" && ac[i].PortName == portName) {
			out = append(out, ac[i])
		}
	}
	return
}

func parseAgentAnnotation(key, value string, pod *v1.Pod) (*AnnotationConfig, error) {
	groups := annotationConfigRegexp.FindStringSubmatch(key)
	if groups[0] == "" {
		return nil, fmt.Errorf("K8s config annotation has invalid agent namespaced key: %s", key)
	}

	conf := &AnnotationConfig{
		AnnotationKey: key,
		Type:          groups[1],
		ConfigKey:     groups[3],
		Value:         value,
	}

	portStr := groups[2]
	if portInt, err := strconv.Atoi(portStr); err != nil {
		conf.PortName = portStr
	} else {
		conf.Port = int32(portInt)
	}

	if conf.Type != "monitorType" && len(conf.ConfigKey) == 0 {
		return nil, fmt.Errorf("K8s config annotation %s is missing a config key", key)
	}
	if conf.Port != 0 && k8sutil.PortByNumber(pod, conf.Port) == nil {
		return nil, fmt.Errorf("K8s config annotation %s references invalid port number %d", key, conf.Port)
	}
	if conf.PortName != "" && k8sutil.PortByName(pod, conf.PortName) == nil {
		return nil, fmt.Errorf("K8s config annotation %s references invalid port name %s", key, conf.PortName)
	}

	return conf, nil
}

func annotationsForPod(pod *v1.Pod) AnnotationConfigs {
	var confs []*AnnotationConfig

	for key, value := range pod.Annotations {
		if !strings.HasPrefix(key, "agent.signalfx.com") {
			continue
		}

		annotationConf, err := parseAgentAnnotation(key, value, pod)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Invalid K8s agent annotation")
			continue
		}

		confs = append(confs, annotationConf)
	}

	return AnnotationConfigs(confs)
}

func configFromAnnotations(
	container string, annotationConfs AnnotationConfigs, pod *v1.Pod, client *k8s.Clientset) (string, map[string]interface{}, error) {

	extraConfig := make(map[string]interface{})
	var monitorType string

	for _, ac := range annotationConfs {
		switch ac.Type {
		case "monitorType":
			monitorType = ac.Value

		case "config":
			extraConfig[ac.ConfigKey] = decodeValueGenerically(strings.TrimSpace(ac.Value))

		case "configFromEnv":
			val, err := k8sutil.EnvValueForContainer(pod, ac.Value, container)
			if err != nil {
				return "", nil, err
			}
			extraConfig[ac.ConfigKey] = decodeValueGenerically(strings.TrimSpace(val))

		case "configFromSecret":
			parts := strings.SplitN(ac.Value, "/", 2)
			if len(parts) != 2 {
				return "", nil, fmt.Errorf("%s value '%s' should be of the form <secretName>/<dataKey>", ac.AnnotationKey, ac.Value)
			}

			secret, err := k8sutil.FetchSecretValue(client, parts[0], parts[1], pod.Namespace)
			if err != nil {
				return "", nil, errors.Wrap(err, "Could not fetch k8s secret")
			}
			// Always treat secret values as strings
			extraConfig[ac.ConfigKey] = secret
		}
	}

	return monitorType, extraConfig, nil
}

// Apply some very basic heuristics to decode string values to the most
// sensible type so that they can be inserted into the endpoint's config map.
// This will then permit them to go through the YAML serialize/deserialized
// cycle and go into the monitor config structs as the right type.
func decodeValueGenerically(val string) interface{} {
	// The literal values of true/false get interpreted as bools
	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}

	// Try to decode as an integer
	if asInt, err := strconv.Atoi(val); err == nil {
		return asInt
	}

	// See if it's an array/list
	if strings.HasPrefix(val, "[") {
		var out []interface{}
		if err := yaml.Unmarshal([]byte(val), &out); err == nil {
			return out
		}
	}

	// Next try to see if it's some kind of object and return the generic
	// yaml MapSlice so that it will be reserialized back to the original form
	// when injected to a monitor instance.  That way we don't have to have
	// knowledge about monitor config types here.
	if strings.HasPrefix(val, "{") {
		var out yaml.MapSlice
		if err := yaml.Unmarshal([]byte(val), &out); err == nil {
			return out
		}
	}

	// Otherwise just treat it as the string it always was
	return val
}
