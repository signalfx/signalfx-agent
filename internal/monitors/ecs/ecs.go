// Package ecs contains a monitor for getting metrics about containers running
// in a docker engine.
package ecs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	dtypes "github.com/docker/docker/api/types"
	dcontainer "github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/core/common/ecs"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	dmonitor "github.com/signalfx/signalfx-agent/internal/monitors/docker"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	log "github.com/sirupsen/logrus"
)

const monitorType = "ecs-metadata"

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig           `acceptsEndpoints:"false"`
	dmonitor.EnhancedMetricsConfig `yaml:",inline"`

	// The URL of the ECS task metadata. Default is http://169.254.170.2/v2/metadata, which is hardcoded by AWS for version 2.
	MetadataEndpoint string `yaml:"metadataEndpoint" default:"http://169.254.170.2/v2/metadata"`
	// The URL of the ECS container stats. Default is http://169.254.170.2/v2/stats, which is hardcoded by AWS for version 2.
	StatsEndpoint string `yaml:"statsEndpoint" default:"http://169.254.170.2/v2/stats"`
	// The maximum amount of time to wait for API requests
	TimeoutSeconds int `yaml:"timeoutSeconds" default:"5"`
	// A mapping of container label names to dimension names. The corresponding
	// label values will become the dimension value for the mapped name.  E.g.
	// `io.kubernetes.container.name: container_spec_name` would result in a
	// dimension called `container_spec_name` that has the value of the
	// `io.kubernetes.container.name` container label.
	LabelsToDimensions map[string]string `yaml:"labelsToDimensions"`
	// A list of filters of images to exclude.  Supports literals, globs, and
	// regex.
	ExcludedImages []string `yaml:"excludedImages"`
}

// Monitor for ECS Metadata
type Monitor struct {
	Output         types.Output
	cancel         func()
	client         *http.Client
	conf           *Config
	ctx            context.Context
	timeout        time.Duration
	taskDimensions map[string]string
	containers     map[string]ecs.Container
	// shouldIgnore - key : container docker id, tells if stats for the container should be ignored.
	// Usually the container was filtered out by excludedImages
	// or container metadata is not received.
	shouldIgnore map[string]bool
	imageFilter  filter.StringFilter
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var err error
	m.imageFilter, err = filter.NewOverridableStringFilter(conf.ExcludedImages)
	if err != nil {
		return errors.Wrapf(err, "Could not load excluded image filter")
	}

	m.conf = conf
	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second
	m.client = &http.Client{
		Timeout: m.timeout,
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())

	isRegistered := false

	utils.RunOnInterval(m.ctx, func() {
		if !isRegistered {
			task, err := fetchTaskMetadata(m.client, m.conf.MetadataEndpoint)
			if err != nil {
				logger.WithFields(log.Fields{
					"error": err,
				}).Error("Could not receive ECS Task Metadata")
				return
			}
			m.taskDimensions = task.GetDimensions()
			m.containers, m.shouldIgnore = parseContainers(task, m.imageFilter)

			isRegistered = true
		}

		m.fetchStatsForAll()
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Fetch a container with given container docker ID and load it to the monitor
// If the container is successfully received, return true. Else, return false
func (m *Monitor) fetchContainer(dockerID string) (ecs.Container, error) {
	body, err := getMetadata(m.client, getURI(m.conf.MetadataEndpoint, dockerID))
	if err != nil {
		return ecs.Container{}, err
	}

	var container ecs.Container

	if err := json.Unmarshal(body, &container); err != nil {
		return ecs.Container{}, errors.Wrapf(err, "Could not parse ecs container json")
	}

	if (m.imageFilter != nil && m.imageFilter.Matches(container.Image)) ||
		container.Type != "NORMAL" {
		return ecs.Container{}, errors.New("The container has been excluded by image filter")
	}

	return container, nil
}

func (m *Monitor) fetchStatsForAll() {
	body, err := getMetadata(m.client, m.conf.StatsEndpoint)

	if err != nil {
		logger.WithError(err).Error("Failed to read ECS stats")
		return
	}

	var stats map[string]dtypes.StatsJSON

	if err := json.Unmarshal(body, &stats); err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Could not parse stats json")
		return
	}

	for dockerID := range stats {
		if m.shouldIgnore[dockerID] {
			continue
		}

		container, ok := m.containers[dockerID]
		if !ok {
			logger.Debugf("Container not found for id %s. Fetching...", dockerID)
			if container, err = m.fetchContainer(dockerID); err != nil {
				m.shouldIgnore[dockerID] = true
				continue
			}
			m.containers[dockerID] = container
		}

		containerJSON := &dtypes.ContainerJSON{
			ContainerJSONBase: &dtypes.ContainerJSONBase{
				ID:   dockerID,
				Name: container.Name,
			},
			Config: &dcontainer.Config{
				Image:    container.Image,
				Hostname: container.Networks[0].IPAddresses[0],
			},
		}
		containerStat := stats[dockerID]
		enhancedMetricsConfig := dmonitor.EnhancedMetricsConfig{
			EnableExtraBlockIOMetrics: m.conf.EnableExtraBlockIOMetrics,
			EnableExtraCPUMetrics:     m.conf.EnableExtraCPUMetrics,
			EnableExtraMemoryMetrics:  m.conf.EnableExtraMemoryMetrics,
			EnableExtraNetworkMetrics: m.conf.EnableExtraNetworkMetrics,
		}
		dps, err := dmonitor.ConvertStatsToMetrics(containerJSON, &containerStat, enhancedMetricsConfig)

		if err != nil {
			logger.WithError(err).Errorf("Could not convert docker stats for container id %s", dockerID)
			return
		}

		dps = append(dps, getTaskLimitMetrics(container, enhancedMetricsConfig)...)

		for i := range dps {
			// Add task metadata to dimensions
			for dimName, v := range m.taskDimensions {
				dps[i].Dimensions[dimName] = v
			}
			for k, dimName := range m.conf.LabelsToDimensions {
				if v := m.containers[dockerID].Labels[k]; v != "" {
					dps[i].Dimensions[dimName] = v
				}
			}
			m.Output.SendDatapoint(dps[i])
		}

		containerProps := &types.DimProperties{
			Dimension: types.Dimension{
				Name:  "container_name",
				Value: container.Name,
			},
			Properties: map[string]string{"known_status": container.KnownStatus},
			Tags:       nil,
		}
		m.Output.SendDimensionProps(containerProps)
	}
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}

func getMetadata(client *http.Client, endpoint string) ([]byte, error) {
	response, err := client.Get(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not connect to %s", endpoint)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Could not connect to %s : %s ", endpoint, http.StatusText(response.StatusCode)))
	}

	body, err := ioutil.ReadAll(response.Body)
	return body, err
}

func fetchTaskMetadata(client *http.Client, metadataEndpoint string) (ecs.TaskMetadata, error) {
	body, err := getMetadata(client, metadataEndpoint)
	if err != nil {
		return ecs.TaskMetadata{}, errors.Wrapf(err, "Failed to read ECS container data")
	}

	task := ecs.TaskMetadata{}
	if err := json.Unmarshal(body, &task); err != nil {
		return ecs.TaskMetadata{}, errors.Wrapf(err, "Could not parse ecs metadata json")
	}

	return task, nil
}

// Fetch all containers in a task
func parseContainers(task ecs.TaskMetadata, imageFilter filter.StringFilter) (map[string]ecs.Container, map[string]bool) {
	containers := map[string]ecs.Container{}
	shouldIgnore := map[string]bool{}

	for i := range task.Containers {
		if (imageFilter == nil ||
			!imageFilter.Matches(task.Containers[i].Image)) &&
			// Containers that are specified in the task definition are of type NORMAL. This will filter out all AWS internal containers
			task.Containers[i].Type == "NORMAL" {
			containers[task.Containers[i].DockerID] = task.Containers[i]
			shouldIgnore[task.Containers[i].DockerID] = false
		} else {
			shouldIgnore[task.Containers[i].DockerID] = true
		}
	}

	return containers, shouldIgnore
}

// Generate datapoints for ECS Task Limits.
func getTaskLimitMetrics(container ecs.Container, enhancedMetricsConfig dmonitor.EnhancedMetricsConfig) []*datapoint.Datapoint {
	var taskLimitDps []*datapoint.Datapoint

	if enhancedMetricsConfig.EnableExtraCPUMetrics {
		cpuDp := sfxclient.Gauge("cpu.limit", nil, container.Limits.CPU)

		cpuDp.Dimensions = map[string]string{}
		cpuDp.Dimensions["plugin"] = "ecs"
		name := strings.TrimPrefix(container.Name, "/")
		cpuDp.Dimensions["container_name"] = name
		cpuDp.Dimensions["plugin_instance"] = name
		cpuDp.Dimensions["container_image"] = container.Image
		cpuDp.Dimensions["container_id"] = container.DockerID
		cpuDp.Dimensions["container_hostname"] = container.Networks[0].IPAddresses[0]

		taskLimitDps = append(taskLimitDps, cpuDp)
	}

	return taskLimitDps
}

func getURI(endpoint string, resourceID string) string {
	queryIdx := strings.Index(endpoint, "?")
	if queryIdx == -1 {
		return fmt.Sprintf("%s/%s", endpoint, resourceID)
	}

	return fmt.Sprintf("%s/%s?%s", endpoint[:queryIdx], resourceID, endpoint[queryIdx+1:])
}
