package cadvisor

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/plugins/monitors/cadvisor/poller"
	"github.com/signalfx/neo-agent/secrets"
	"github.com/signalfx/neo-agent/utils"
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
	lock    sync.Mutex
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
	var labels [][]string
	c.Config.UnmarshalKey("excludedLabels", labels)
	for _, label := range labels {
		var kcomp *regexp.Regexp
		var vcomp *regexp.Regexp
		var value = ".*"
		var err error
		if len(label) >= 1 {
			if kcomp, err = regexp.Compile(label[0]); err != nil {
				log.Printf("Unable to compile regex pattern '%s' for label: '%v'", label[0], err)
				continue
			}
		} else {
			// this is probably a bug if it is ever encountered
			log.Printf("Unable to compile regex pattern because label criteria was empty.")
			continue
		}

		if len(label) == 2 {
			if label[1] != "" {
				value = label[1]
			}
		}
		if vcomp, err = regexp.Compile(value); err != nil {
			log.Printf("Unable to compile regex pattern '%s' for label {'%s' : '%s'}: '%v'", value, label[0], value, err)
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
			log.Printf("Unable to compile regex pattern '%s' for image: '%v'", image, err)
		} else {
			eximages = append(eximages, comp)
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
			log.Printf("Unable to copmile regex pattern '%s' for name: '%v'", name, err)
		} else {
			exnames = append(exnames, comp)
		}
	}
	return exnames
}

func (c *Cadvisor) getMetricFilter() map[string]bool {
	var filters = c.Config.GetStringSlice("excludedMetrics")

	// convert the config into a map so we don't have to iterate over and over
	var filterMap = utils.StringSliceToMap(filters)

	return filterMap
}

// Configure and start/restart cadvisor plugin
func (c *Cadvisor) Configure(config *viper.Viper) error {
	// Lock for reconfiguring the plugin
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop if cadvisor was previously running
	c.Stop()

	c.Config = config
	c.stop = nil
	c.stopped = nil

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
		ExcludedMetrics:        c.getMetricFilter(),
	}

	if c.stop, c.stopped, err = poller.MonitorNode(cfg, forwarder, time.Duration(dataSendRate)*time.Second); err != nil {
		log.Printf("monitoring cadvisor node failed: %s", err)
	}

	log.Print("started cadvisor monitoring")

	return nil
}

// Stop cadvisor plugin
func (c *Cadvisor) Stop() {
	// tell cadvisor to stop
	if c.stop != nil {
		c.stop <- true
	}
	// read the stopped signal from cadvisor
	if c.stopped != nil {
		<-c.stopped
	}
}
