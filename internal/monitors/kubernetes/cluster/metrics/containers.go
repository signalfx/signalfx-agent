package metrics

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
	v1 "k8s.io/api/core/v1"
)

func stripContainerIDPrefix(id string) string {
	out := strings.Replace(id, "docker://", "", 1)
	out = strings.Replace(out, "cri-o://", "", 1)

	return out
}

func datapointsForContainerStatus(cs v1.ContainerStatus, contDims map[string]string) []*datapoint.Datapoint {
	dps := []*datapoint.Datapoint{
		datapoint.New(
			"kubernetes.container_restart_count",
			contDims,
			datapoint.NewIntValue(int64(cs.RestartCount)),
			datapoint.Gauge,
			time.Now()),
		datapoint.New(
			"kubernetes.container_ready",
			contDims,
			datapoint.NewIntValue(int64(utils.BoolToInt(cs.Ready))),
			datapoint.Gauge,
			time.Now()),
	}

	return dps
}

func datapointsForContainerSpec(c v1.Container, contDims map[string]string) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint

	if val, ok := c.Resources.Requests[v1.ResourceCPU]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_cpu_request",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	if val, ok := c.Resources.Limits[v1.ResourceCPU]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_cpu_limit",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	if val, ok := c.Resources.Requests[v1.ResourceMemory]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_memory_request",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	if val, ok := c.Resources.Limits[v1.ResourceMemory]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_memory_limit",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	if val, ok := c.Resources.Requests[v1.ResourceEphemeralStorage]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_ephemeral_storage_request",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	if val, ok := c.Resources.Limits[v1.ResourceEphemeralStorage]; ok {
		dps = append(dps,
			datapoint.New(
				"kubernetes.container_ephemeral_storage_limit",
				contDims,
				datapoint.NewIntValue(val.Value()),
				datapoint.Gauge,
				time.Now()))
	}

	return dps
}

func getAllContainerDimensions(id string, name string, image string, dims map[string]string) map[string]string {
	out := utils.CloneStringMap(dims)

	out["container_id"] = stripContainerIDPrefix(id)
	out["container_spec_name"] = name
	out["container_image"] = image

	return out
}
