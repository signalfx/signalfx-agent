package cadvisor

import (
	"regexp"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/monitors/cadvisor/converter"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

// Config that is specific to the cAdvisor monitor
type Config struct {
	ExcludedLabels  [][]string `yaml:"excludedLabels"`
	ExcludedImages  []string   `yaml:"excludedImages"`
	ExcludedNames   []string   `yaml:"excludedNames"`
	ExcludedMetrics []string   `yaml:"excludedMetrics"`
}

// Monitor pulls metrics from a cAdvisor-compatible endpoint
type Monitor struct {
	monConfig *config.MonitorConfig
	lock      sync.Mutex
	stop      chan bool
	stopped   chan bool

	excludedNames   []*regexp.Regexp
	excludedImages  []*regexp.Regexp
	excludedLabels  [][]*regexp.Regexp
	excludedMetrics map[string]bool

	AgentMeta *meta.AgentMeta
}

func (m *Monitor) getLabelFilter(labels [][]string) [][]*regexp.Regexp {
	var exlabels = [][]*regexp.Regexp{}
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

func (m *Monitor) getImageFilter(images []string) []*regexp.Regexp {
	var eximages = []*regexp.Regexp{}
	for _, image := range images {
		if comp, err := regexp.Compile(image); err != nil {
			log.Printf("Unable to compile regex pattern '%s' for image: '%v'", image, err)
		} else {
			eximages = append(eximages, comp)
		}
	}
	return eximages
}

func (m *Monitor) getNameFilter(names []string) []*regexp.Regexp {
	var exnames = []*regexp.Regexp{}
	for _, name := range names {
		if comp, err := regexp.Compile(name); err != nil {
			log.Printf("Unable to compile regex pattern '%s' for name: '%v'", name, err)
		} else {
			exnames = append(exnames, comp)
		}
	}
	return exnames
}

func (m *Monitor) getMetricFilter(filters []string) map[string]bool {
	// convert the config into a map so we don't have to iterate over and over
	var filterMap = utils.StringSliceToMap(filters)

	return filterMap
}

// Configure and start/restart cadvisor plugin
func (m *Monitor) Configure(conf *Config, monConfig *config.MonitorConfig, sendDP func(*datapoint.Datapoint), statProvider converter.InfoProvider) error {
	// Lock for reconfiguring the plugin
	m.lock.Lock()
	defer m.lock.Unlock()

	m.monConfig = monConfig

	m.stop = nil
	m.stopped = nil

	m.excludedImages = m.getImageFilter(conf.ExcludedImages)
	m.excludedNames = m.getNameFilter(conf.ExcludedNames)
	m.excludedLabels = m.getLabelFilter(conf.ExcludedLabels)
	m.excludedMetrics = m.getMetricFilter(conf.ExcludedMetrics)

	collector := converter.NewCadvisorCollector(statProvider, sendDP, m.AgentMeta.Hostname, monConfig.ExtraDimensions, m.excludedImages, m.excludedNames, m.excludedLabels, m.excludedMetrics)

	m.stop, m.stopped = monitorNode(monConfig.IntervalSeconds, collector)

	return nil
}

func monitorNode(intervalSeconds int, collector *converter.CadvisorCollector) (stop chan bool, stopped chan bool) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	stop = make(chan bool, 1)
	stopped = make(chan bool, 1)

	go func() {
		collector.Collect()
		for {
			select {
			case <-stop:
				log.Info("Stopping cAdvisor collection")
				ticker.Stop()
				stopped <- true
				return
			case <-ticker.C:
				collector.Collect()
			}
		}
	}()

	return stop, stopped
}

// Shutdown cadvisor plugin
func (m *Monitor) Shutdown() {
	// tell cadvisor to stop
	if m.stop != nil {
		close(m.stop)
	}
}
