package cadvisor

import (
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/signalfx/neo-agent/core/config"
	"github.com/signalfx/neo-agent/monitors"
	"github.com/signalfx/neo-agent/monitors/cadvisor/poller"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

const (
	_type = "monitors/cadvisor"
)

var logger = log.WithFields(log.Fields{"monitorType": _type})

func init() {
	monitors.Register(_type, func() interface{} { return &Cadvisor{} }, &Config{})
}

type Config struct {
	config.MonitorConfig
	CAdvisorURL     string
	ExcludedLabels  [][]string
	ExcludedImages  []string
	ExcludedNames   []string
	ExcludedMetrics []string

	KubernetesAPI struct {
		AuthType       string `default:"serviceAccount"`
		ClientCertPath string
		ClientKeyPath  string
		CACertPath     string `yaml:"caCertPath,omitempty"`
	} `yaml:"kubernetesAPI"`
}

// Cadvisor plugin struct
type Cadvisor struct {
	config  *Config
	lock    sync.Mutex
	stop    chan bool
	stopped chan bool
}

func (c *Cadvisor) getLabelFilter() [][]*regexp.Regexp {
	var exlabels = [][]*regexp.Regexp{}
	for _, label := range c.config.ExcludedLabels {
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

func (c *Cadvisor) getImageFilter() []*regexp.Regexp {
	var eximages = []*regexp.Regexp{}
	for _, image := range c.config.ExcludedImages {
		if comp, err := regexp.Compile(image); err != nil {
			log.Printf("Unable to compile regex pattern '%s' for image: '%v'", image, err)
		} else {
			eximages = append(eximages, comp)
		}
	}
	return eximages
}

func (c *Cadvisor) getNameFilter() []*regexp.Regexp {
	var exnames = []*regexp.Regexp{}
	for _, name := range c.config.ExcludedNames {
		if comp, err := regexp.Compile(name); err != nil {
			log.Printf("Unable to copmile regex pattern '%s' for name: '%v'", name, err)
		} else {
			exnames = append(exnames, comp)
		}
	}
	return exnames
}

func (c *Cadvisor) getMetricFilter() map[string]bool {
	var filters = c.config.ExcludedMetrics

	// convert the config into a map so we don't have to iterate over and over
	var filterMap = utils.StringSliceToMap(filters)

	return filterMap
}

// Configure and start/restart cadvisor plugin
func (c *Cadvisor) Configure(conf *Config) bool {
	// Lock for reconfiguring the plugin
	c.lock.Lock()
	defer c.lock.Unlock()

	// Stop if cadvisor was previously running
	c.Shutdown()

	c.stop = nil
	c.stopped = nil

	dimensions := conf.ExtraDimensions
	clusterName := conf.ExtraDimensions["kubernetes_cluster"]
	forwarder := poller.NewSfxClient(conf.IngestURL.String(), conf.SignalFxAccessToken)
	cfg := &poller.Config{
		IngestURL:              conf.IngestURL.String(),
		APIToken:               conf.SignalFxAccessToken,
		DataSendRate:           strconv.Itoa(conf.IntervalSeconds),
		ClusterName:            clusterName,
		NodeServiceRefreshRate: "",
		CadvisorPort:           0,
		KubernetesURL:          "",
		KubernetesUsername:     "",
		CadvisorURL:            []string{conf.CAdvisorURL},
		KubernetesPassword:     "",
		DefaultDimensions:      dimensions,
		ExcludedImages:         c.getImageFilter(),
		ExcludedNames:          c.getNameFilter(),
		ExcludedLabels:         c.getLabelFilter(),
		ExcludedMetrics:        c.getMetricFilter(),
	}

	var err error
	if c.stop, c.stopped, err = poller.MonitorNode(cfg, forwarder, time.Duration(conf.IntervalSeconds)*time.Second); err != nil {
		log.Errorf("monitoring cadvisor node failed: %s", err)
		return false
	}

	c.config = conf

	return true
}

// Shutdown cadvisor plugin
func (c *Cadvisor) Shutdown() {
	// tell cadvisor to stop
	if c.stop != nil {
		c.stop <- true
	}
	// read the stopped signal from cadvisor
	if c.stopped != nil {
		<-c.stopped
	}
}
