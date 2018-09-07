package docker

import (
	"regexp"
	"strings"

	"github.com/docker/go-connections/nat"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

var labelConfigRegexp = regexp.MustCompile(
	`^agent.signalfx.com\.` +
		`(?P<type>monitorType|config)` +
		`\.(?P<port>[\w]+)(?:-(?P<port_name>[\w]+))?` +
		`(?:\.(?P<config_key>\w+))?$`)

type labelConfig struct {
	MonitorType   string
	Configuration map[string]interface{}
}

type contPort struct {
	nat.Port
	Name string
}

func getConfigLabels(labels map[string]string) map[contPort]*labelConfig {
	portMap := map[contPort]*labelConfig{}

	for k, v := range labels {
		if !strings.HasPrefix(k, "agent.signalfx.com") {
			continue
		}

		groups := utils.RegexpGroupMap(labelConfigRegexp, k)
		if groups == nil {
			logger.Errorf("Docker label has invalid agent namespaced key: %s", k)
			continue
		}

		natPort, err := nat.NewPort(nat.SplitProtoPort(groups["port"]))
		if err != nil {
			logger.WithError(err).Errorf("Docker label port '%s' could not be parsed", groups["port"])
			continue
		}

		portObj := contPort{
			Port: natPort,
			Name: groups["port_name"],
		}

		if _, ok := portMap[portObj]; !ok {
			portMap[portObj] = &labelConfig{
				Configuration: map[string]interface{}{},
			}
		}

		if groups["type"] == "monitorType" {
			portMap[portObj].MonitorType = v
		} else {
			portMap[portObj].Configuration[groups["config_key"]] = utils.DecodeValueGenerically(v)
		}
	}

	return portMap
}
