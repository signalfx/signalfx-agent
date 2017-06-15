package cadvisor

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/signalfx/cadvisor-integration/poller"
	"github.com/signalfx/neo-agent/config"
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

// getLabelFilter - parses viper config and returns label filter
func (c *Cadvisor) getLabelFilter() [][]*regexp.Regexp {
	var exlabels = [][]*regexp.Regexp{}
	var labels []*config.Label
	c.Config.UnmarshalKey("excludedLabels", &labels)
	for _, label := range labels {
		var kcomp *regexp.Regexp
		var vcomp *regexp.Regexp
		var value = ".*"
		var err error
		if kcomp, err = regexp.Compile(label.Key); err != nil {
			log.Println(fmt.Sprintf("Unable to compile regex pattern '%s' for label '%s' : '%s'}", label.Key, label.Key, label.Value))
			continue
		}
		if label.Value != "" {
			value = label.Value
		}
		if vcomp, err = regexp.Compile(value); err != nil {
			log.Println(fmt.Sprintf("Unable to compile regex pattern '%s' for label '%s' : '%s'}", value, label.Key, value))
			continue
		}
		exlabels = append(exlabels, []*regexp.Regexp{kcomp, vcomp})
	}

	return exlabels
}

// getImageFilter - parses viper config and returns image filter
func (c *Cadvisor) getImageFilter() []*regexp.Regexp {
	var eximages = []*regexp.Regexp{}
	images := c.Config.GetStringSlice("excludedImages")
	for _, image := range images {
		if comp, err := regexp.Compile(image); err != nil {
			eximages = append(eximages, comp)
		} else {
			log.Println(fmt.Sprintf("Unable to compile regex pattern '%s' for image", image))
		}
	}
	return eximages
}

// getNameFilter - parses viper config and returns name filter
func (c *Cadvisor) getNameFilter() []*regexp.Regexp {
	var exnames = []*regexp.Regexp{}
	names := c.Config.GetStringSlice("excludedNames")
	for _, name := range names {
		if comp, err := regexp.Compile(name); err != nil {
			exnames = append(exnames, comp)
		} else {
			log.Println(fmt.Sprintf("Unable to copmile regex pattern '%s' for name", name))
		}
	}
	return exnames
}

// Start cadvisor plugin
func (c *Cadvisor) Start() error {
	apiToken, err := secrets.EnvSecret("SFX_ACCESS_TOKEN")
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
		ExcludedImages:         c.getImageFilter(),
		ExcludedNames:          c.getNameFilter(),
		ExcludedLabels:         c.getLabelFilter(),
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
