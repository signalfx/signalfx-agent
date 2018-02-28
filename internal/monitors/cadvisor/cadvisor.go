package cadvisor

import (
	"time"

	"github.com/google/cadvisor/client"
	info "github.com/google/cadvisor/info/v1"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
)

const (
	cadvisorType = "cadvisor"
)

// MONITOR(cadvisor): This monitor pulls metrics directly from cadvisor, which
// conventionally runs on port 4194, but can be configured to anything.  If you
// are running on Kubernetes, consider the [kubelet-stats](./kubelet-stats.md)
// monitor because many K8s nodes do not expose cAdvisor on a network port,
// even though they are running it within Kubelet.
//
// If you are running containers with Docker, there is a fair amount of
// duplication with the `collectd/docker` monitor in terms of the metrics sent
// (under distinct metric names) so you may want to consider not enabling the
// Docker monitor in a K8s environment, or else use filtering to whitelist only
// certain metrics.  Note that this will cause the built-in Docker dashboards
// to be blank, but container metrics will be available on the Kubernetes
// dashboards instead.

func init() {
	monitors.Register(cadvisorType, func() interface{} { return &Cadvisor{} }, &CHTTPConfig{})
}

// CHTTPConfig is the monitor-specific config for cAdvisor
type CHTTPConfig struct {
	config.MonitorConfig
	// Where to find cAdvisor
	CAdvisorURL string `yaml:"cadvisorURL" default:"http://localhost:4194"`
}

// Cadvisor is the monitor that goes straight to the exposed cAdvisor port to
// get metrics
type Cadvisor struct {
	Monitor
	Output types.Output
}

// Configure the cAdvisor monitor
func (c *Cadvisor) Configure(conf *CHTTPConfig) error {
	cadvisorClient, err := client.NewClient(conf.CAdvisorURL)
	if err != nil {
		return errors.Wrap(err, "Could not create cAdvisor client")
	}

	return c.Monitor.Configure(&conf.MonitorConfig, c.Output.SendDatapoint, newCadvisorInfoProvider(cadvisorClient))
}

type cadvisorInfoProvider struct {
	cc         *client.Client
	lastUpdate time.Time
}

func (cip *cadvisorInfoProvider) SubcontainersInfo(containerName string) ([]info.ContainerInfo, error) {
	curTime := time.Now()
	info, err := cip.cc.AllDockerContainers(&info.ContainerInfoRequest{Start: cip.lastUpdate, End: curTime})
	if len(info) > 0 {
		cip.lastUpdate = curTime
	}
	return info, err
}

func (cip *cadvisorInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return cip.cc.MachineInfo()
}

func newCadvisorInfoProvider(cadvisorClient *client.Client) *cadvisorInfoProvider {
	return &cadvisorInfoProvider{
		cc:         cadvisorClient,
		lastUpdate: time.Now(),
	}
}
