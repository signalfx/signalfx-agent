package metrics

import (
	"strings"

	atypes "github.com/signalfx/signalfx-agent/internal/monitors/types"
	v1 "k8s.io/api/core/v1"
)

func dimPropsForContainer(cs v1.ContainerStatus) *atypes.DimProperties {

	containerProps := make(map[string]string)

	if cs.State.Running != nil {
		containerProps["container_status"] = "running"
	}

	if cs.State.Terminated != nil {
		containerProps["container_status"] = "terminated"
		containerProps["container_status_reason"] = cs.State.Terminated.Reason
	}

	if cs.State.Waiting != nil {
		containerProps["container_status"] = "waiting"
		containerProps["container_status_reason"] = cs.State.Waiting.Reason
	}

	if len(containerProps) > 0 {
		return &atypes.DimProperties{
			Dimension: atypes.Dimension{
				Name:  "container_id",
				Value: strings.Replace(cs.ContainerID, "docker://", "", 1),
			},
			Properties: containerProps,
		}
	}
	return nil
}
