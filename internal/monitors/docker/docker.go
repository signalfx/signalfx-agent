// Package docker contains a monitor for getting metrics about containers running
// in a docker engine.
package docker

import (
	"context"
	"strings"
	"sync"
	"time"

	dtypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/pkg/errors"
	dockercommon "github.com/signalfx/signalfx-agent/internal/core/common/docker"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/filter"
	log "github.com/sirupsen/logrus"
)

const monitorType = "docker-container-stats"
const dockerAPIVersion = "v1.22"

// MONITOR(docker-container-stats): This monitor reads container stats from a
// Docker API server.  It is meant as a metric-compatible replacement of our
// [docker-collectd](https://github.com/signalfx/docker-collectd-plugin)
// plugin, which scales rather poorly against a large number of containers.
//
// This currently does not support CPU share/quota metrics.
//
// If you are running the agent directly on a host (outside of a container
// itself) and you are using the default Docker UNIX socket URL, you will
// probably need to add the `signalfx-agent` user to the `docker` group in
// order to have permission to access the Docker API via the socket.
//
// Requires Docker API version 1.22+.

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`

	// The URL of the docker server
	DockerURL string `yaml:"dockerURL" default:"unix:///var/run/docker.sock"`
	// The maximum amount of time to wait for docker API requests
	TimeoutSeconds int `yaml:"timeoutSeconds" default:"5"`
	// A mapping of container label names to dimension names. The corresponding
	// label values will become the dimension value for the mapped name.  E.g.
	// `io.kubernetes.container.name: container_spec_name` would result in a
	// dimension called `container_spec_name` that has the value of the
	// `io.kubernetes.container.name` container label.
	LabelsToDimensions map[string]string `yaml:"labelsToDimensions"`
	// A mapping of container environment variable names to dimension
	// names.  The corresponding env var values become the dimension values on
	// the emitted metrics.  E.g. `APP_VERSION: version` would result in
	// datapoints having a dimension called `version` whose value is the value
	// of the `APP_VERSION` envvar configured for that particular container, if
	// present.
	EnvToDimensions map[string]string `yaml:"envToDimensions"`
	// A list of filters of images to exclude.  Supports literals, globs, and
	// regex.
	ExcludedImages []string `yaml:"excludedImages"`
}

// Monitor for Docker
type Monitor struct {
	Output  types.Output
	cancel  func()
	ctx     context.Context
	client  *docker.Client
	timeout time.Duration
}

type dockerContainer struct {
	*dtypes.ContainerJSON
	EnvMap map[string]string
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	defaultHeaders := map[string]string{"User-Agent": "signalfx-agent"}

	var err error
	m.client, err = docker.NewClient(conf.DockerURL, dockerAPIVersion, nil, defaultHeaders)
	if err != nil {
		return errors.Wrapf(err, "Could not create docker client")
	}

	m.timeout = time.Duration(conf.TimeoutSeconds) * time.Second

	m.ctx, m.cancel = context.WithCancel(context.Background())

	imageFilter, err := filter.NewBasicStringFilter(conf.ExcludedImages)
	if err != nil {
		return err
	}

	lock := sync.Mutex{}
	containers := map[string]dockerContainer{}
	isRegistered := false

	changeHandler := func(old *dtypes.ContainerJSON, new *dtypes.ContainerJSON) {
		if old == nil && new == nil {
			return
		}

		var id string
		if new != nil {
			id = new.ID
		} else {
			id = old.ID
		}

		lock.Lock()
		defer lock.Unlock()

		if new == nil || (!new.State.Running || new.State.Paused) {
			logger.Debugf("Container %s is no longer running", id)
			delete(containers, id)
			return
		}
		logger.Infof("Monitoring docker container %s", id)
		containers[id] = dockerContainer{
			ContainerJSON: new,
			EnvMap:        parseContainerEnvSlice(new.Config.Env),
		}
	}

	utils.RunOnInterval(m.ctx, func() {
		// Repeat the watch setup in the face of errors in case the docker
		// engine is non-responsive when the monitor starts.
		if !isRegistered {
			var err error
			err = dockercommon.ListAndWatchContainers(m.ctx, m.client, changeHandler, imageFilter, logger)
			if err != nil {
				logger.WithError(err).Error("Could not list docker containers")
				return
			}
			isRegistered = true
		}

		// Individual container objects don't need to be protected by the lock,
		// only the map that holds them.
		lock.Lock()
		for id := range containers {
			go m.fetchStats(containers[id], conf.LabelsToDimensions, conf.EnvToDimensions)
		}
		lock.Unlock()

	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Instead of streaming stats like the collectd plugin does, fetch the stats in
// parallel in individual goroutines.  This is much easier on CPU usage since
// we aren't doing something every second across all containers, but only
// something once every metric interval.
func (m *Monitor) fetchStats(container dockerContainer, labelMap map[string]string, envMap map[string]string) {
	ctx, cancel := context.WithTimeout(m.ctx, m.timeout)
	stats, err := m.client.ContainerStats(ctx, container.ID, false)
	if err != nil {
		cancel()
		logger.WithError(err).Errorf("Could not fetch docker stats for container id %s", container.ID)
		return
	}

	dps, err := convertStatsToMetrics(container.ContainerJSON, &stats)
	cancel()
	if err != nil {
		logger.WithError(err).Errorf("Could not convert docker stats for container id %s", container.ID)
		return
	}

	for i := range dps {
		for k, dimName := range envMap {
			if v := container.EnvMap[k]; v != "" {
				dps[i].Dimensions[dimName] = v
			}
		}
		for k, dimName := range labelMap {
			if v := container.Config.Labels[k]; v != "" {
				dps[i].Dimensions[dimName] = v
			}
		}
		m.Output.SendDatapoint(dps[i])
	}
}

func parseContainerEnvSlice(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, v := range env {
		parts := strings.Split(v, "=")
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
