package cadvisor

import (
	"errors"
	"log"
	"time"

	"strconv"

	"github.com/signalfx/cadvisor-integration/poller"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/secrets"
	"github.com/spf13/viper"
)

const (
	pluginType = "monitors/cadvisor"
)

func init() {
	plugins.Register(pluginType, NewCadvisor)
}

// Cadvisor plugin struct
type Cadvisor struct {
	plugins.Plugin
	stop    chan bool
	stopped chan bool
}

// NewCadvisor creates a new plugin instance
func NewCadvisor(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}
	return &Cadvisor{Plugin: plugin}, nil
}

// Start cadvisor plugin
func (c *Cadvisor) Start() error {
	apiToken, err := secrets.EnvSecret("SFX_API_TOKEN")
	if err != nil {
		return err
	}

	ingestURL := viper.GetString("ingesturl")
	if ingestURL == "" {
		return errors.New("ingestURL cannot be empty")
	}

	dataSendRate := c.Config.GetInt("dataSendRate")
	if dataSendRate <= 0 {
		return errors.New("dataSendRate cannot be zero or less")
	}

	cadvisorURL := c.Config.GetString("cadvisorurl")
	if cadvisorURL == "" {
		return errors.New("cadvisorURL cannot be empty")
	}

	dimensions := viper.GetStringMapString("dimensions")
	clusterName := dimensions["kubernetes_cluster"]

	forwarder := poller.NewSfxClient(ingestURL, apiToken)
	cfg := &poller.Config{
		IngestURL:              ingestURL,
		APIToken:               apiToken,
		DataSendRate:           strconv.Itoa(dataSendRate),
		ClusterName:            clusterName,
		NodeServiceRefreshRate: "",
		CadvisorPort:           0,
		KubernetesURL:          "",
		KubernetesUsername:     "",
		CadvisorURL:            []string{cadvisorURL},
		KubernetesPassword:     "",
		DefaultDimensions:      dimensions,
	}

	if c.stop, c.stopped, err = poller.MonitorNode(cfg, forwarder, time.Duration(dataSendRate)*time.Second); err != nil {
		log.Printf("monitoring cadvisor node failed: %s", err)
	}

	log.Print("started cadvisor monitoring")

	return nil
}

// Stop cadvisor plugin
func (c *Cadvisor) Stop() {
	if c.stop != nil {
		c.stop <- true
	}
}

// Reload cadvisor plugin
func (c *Cadvisor) Reload(config *viper.Viper) error {
	if c.stop != nil {
		c.stop <- true
	}
	if c.stopped != nil {
		<-c.stopped
	}
	c.Config = config
	c.stop = nil
	c.stopped = nil
	return c.Start()
}
