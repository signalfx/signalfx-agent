package cadvisor

import (
	"time"

	"github.com/google/cadvisor/client"
	info "github.com/google/cadvisor/info/v1"
	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
)

const (
	cadvisorType = "cadvisor"
)

func init() {
	monitors.Register(cadvisorType, func() interface{} { return &Cadvisor{} }, &CHTTPConfig{})
}

// CHTTPConfig is the monitor-specific config for cAdvisor
type CHTTPConfig struct {
	config.MonitorConfig
	Config
	CAdvisorURL string `yaml:"cadvisorURL" default:"http://localhost:4194"`
}

// Cadvisor is the monitor that goes straight to the exposed cAdvisor port to
// get metrics
type Cadvisor struct {
	Monitor
	DPs chan<- *datapoint.Datapoint
}

// Configure the cAdvisor monitor
func (c *Cadvisor) Configure(conf *CHTTPConfig) error {
	cadvisorClient, err := client.NewClient(conf.CAdvisorURL)
	if err != nil {
		return errors.Wrap(err, "Could not create cAdvisor client")
	}

	return c.Monitor.Configure(&conf.Config, &conf.MonitorConfig, c.DPs, newCadvisorInfoProvider(cadvisorClient))
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
