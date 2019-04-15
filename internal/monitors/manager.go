package monitors

import (
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/golib/trace"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/core/dpfilters"
	"github.com/signalfx/signalfx-agent/internal/core/meta"
	"github.com/signalfx/signalfx-agent/internal/core/services"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

// MonitorManager coordinates the startup and shutdown of monitors based on the
// configuration provided by the user.  Monitors that have discovery rules can
// be injected with multiple services.  If a monitor does not have a discovery
// rule (a "static" monitor), it will be started immediately (as soon as
// Configure is called).
type MonitorManager struct {
	monitorConfigs map[uint64]config.MonitorCustomConfig
	// Keep track of which services go with which monitor
	activeMonitors []*ActiveMonitor
	badConfigs     map[uint64]*config.MonitorConfig
	lock           sync.Mutex
	// Map of service endpoints that have been discovered
	discoveredEndpoints map[services.ID]services.Endpoint

	DPs            chan<- *datapoint.Datapoint
	Events         chan<- *event.Event
	DimensionProps chan<- *types.DimProperties
	TraceSpans     chan<- *trace.Span

	// TODO: AgentMeta is rather hacky so figure out a better way to share agent
	// metadata with monitors
	agentMeta       *meta.AgentMeta
	intervalSeconds int

	idGenerator func() string
}

// NewMonitorManager creates a new instance of the MonitorManager
func NewMonitorManager(agentMeta *meta.AgentMeta) *MonitorManager {
	return &MonitorManager{
		monitorConfigs:      make(map[uint64]config.MonitorCustomConfig),
		activeMonitors:      make([]*ActiveMonitor, 0),
		badConfigs:          make(map[uint64]*config.MonitorConfig),
		discoveredEndpoints: make(map[services.ID]services.Endpoint),
		idGenerator:         utils.NewIDGenerator(),
		agentMeta:           agentMeta,
	}
}

// Configure receives a list of monitor configurations.  It will start up any
// static monitors and watch discovered services to see if any match dynamic
// monitors.
func (mm *MonitorManager) Configure(confs []config.MonitorConfig, collectdConf *config.CollectdConfig, intervalSeconds int) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	mm.intervalSeconds = intervalSeconds
	for i := range confs {
		confs[i].IntervalSeconds = utils.FirstNonZero(confs[i].IntervalSeconds, intervalSeconds)
	}

	requireSoloTrue := anyMarkedSolo(confs)

	newConfig, deletedHashes := diffNewConfig(confs, mm.allConfigHashes())

	if !collectdConf.DisableCollectd {
		// By configuring collectd with the monitor manager, we absolve the monitor
		// instances of having to know about collectd config, which makes it easier
		// to create monitor config from disparate sources such as from observers.
		if err := collectd.ConfigureMainCollectd(collectdConf); err != nil {
			log.WithFields(log.Fields{
				"error":          err,
				"collectdConfig": spew.Sdump(collectdConf),
			}).Error("Could not configure collectd")
		}
	}

	for _, hash := range deletedHashes {
		mm.deleteMonitorsByConfigHash(hash)

		delete(mm.monitorConfigs, hash)
		delete(mm.badConfigs, hash)
	}

	for i := range newConfig {
		conf := newConfig[i]
		hash := conf.Hash()

		if requireSoloTrue && !conf.Solo {
			log.Infof("Solo mode is active, skipping monitor of type %s", conf.Type)
			continue
		}

		monConfig, err := mm.handleNewConfig(&conf)
		if err != nil {
			log.WithFields(log.Fields{
				"monitorType": conf.Type,
				"error":       err,
			}).Error("Could not process configuration for monitor")
			conf.ValidationError = err.Error()
			mm.badConfigs[hash] = &conf
			continue
		}

		mm.monitorConfigs[hash] = monConfig
	}
}

func (mm *MonitorManager) allConfigHashes() map[uint64]bool {
	hashes := make(map[uint64]bool)
	for h := range mm.monitorConfigs {
		hashes[h] = true
	}
	for h := range mm.badConfigs {
		hashes[h] = true
	}
	return hashes
}

// Returns the any new configs and any removed config hashes
func diffNewConfig(confs []config.MonitorConfig, oldHashes map[uint64]bool) ([]config.MonitorConfig, []uint64) {
	newConfigHashes := make(map[uint64]bool)
	var newConfig []config.MonitorConfig
	for i := range confs {
		hash := confs[i].Hash()
		if !oldHashes[hash] {
			newConfig = append(newConfig, confs[i])
		}

		if newConfigHashes[hash] {
			log.WithFields(log.Fields{
				"monitorType": confs[i].Type,
				"config":      confs[i],
			}).Error("Monitor config is duplicated")
			continue
		}

		newConfigHashes[hash] = true
	}

	var deletedHashes []uint64
	for hash := range oldHashes {
		// If we didn't see it in the latest config slice then we need to
		// delete anything using it.
		if !newConfigHashes[hash] {
			deletedHashes = append(deletedHashes, hash)
		}
	}

	return newConfig, deletedHashes
}

func (mm *MonitorManager) handleNewConfig(conf *config.MonitorConfig) (config.MonitorCustomConfig, error) {
	monConfig, err := getCustomConfigForMonitor(conf)
	if err != nil {
		return nil, err
	}

	if configOnlyAllowsSingleInstance(monConfig) {
		if len(mm.monitorConfigsForType(conf.Type)) > 0 {
			return nil, fmt.Errorf("Monitor type %s only allows a single instance at a time", conf.Type)
		}
	}

	// No discovery rule means that the monitor should run from the start
	if conf.DiscoveryRule == "" {
		return monConfig, mm.createAndConfigureNewMonitor(monConfig, nil)
	}

	mm.makeMonitorsForMatchingEndpoints(monConfig)
	// We need to go and see if any discovered endpoints should be
	// monitored by this config, if they aren't already.
	return monConfig, nil
}

func (mm *MonitorManager) makeMonitorsForMatchingEndpoints(conf config.MonitorCustomConfig) {
	for id, endpoint := range mm.discoveredEndpoints {
		// Self configured endpoints are monitored immediately upon being
		// created and never need to be matched against discovery rules.
		if endpoint.Core().IsSelfConfigured() {
			continue
		}

		log.WithFields(log.Fields{
			"monitorType":   conf.MonitorConfigCore().Type,
			"discoveryRule": conf.MonitorConfigCore().DiscoveryRule,
			"endpoint":      endpoint,
		}).Debug("Trying to find config that matches discovered endpoint")

		if mm.isEndpointIDMonitoredByConfig(conf, id) {
			log.Debug("The monitor is already monitored")
			continue
		}

		if matched, err := mm.monitorEndpointIfRuleMatches(conf, endpoint); matched {
			if err != nil {
				log.WithFields(log.Fields{
					"error":       err,
					"endpointID":  endpoint.Core().ID,
					"monitorType": conf.MonitorConfigCore().Type,
				}).Error("Error monitoring endpoint that matched rule")
			} else {
				log.WithFields(log.Fields{
					"endpointID":  endpoint.Core().ID,
					"monitorType": conf.MonitorConfigCore().Type,
				}).Info("Now monitoring discovered endpoint")
			}
		} else {
			log.Debug("The monitor did not match")
		}
	}
}

func (mm *MonitorManager) isEndpointIDMonitoredByConfig(conf config.MonitorCustomConfig, id services.ID) bool {
	for _, am := range mm.activeMonitors {
		if conf.MonitorConfigCore().Hash() == am.configHash {
			return true
		}
	}
	return false
}

// Returns true is the service did match a rule in this monitor config
func (mm *MonitorManager) monitorEndpointIfRuleMatches(config config.MonitorCustomConfig, endpoint services.Endpoint) (bool, error) {
	if config.MonitorConfigCore().DiscoveryRule == "" || !services.DoesServiceMatchRule(endpoint, config.MonitorConfigCore().DiscoveryRule) {
		return false, nil
	}

	err := mm.createAndConfigureNewMonitor(config, endpoint)
	if err != nil {
		return true, err
	}

	return true, nil
}

// EndpointAdded should be called when a new service is discovered
func (mm *MonitorManager) EndpointAdded(endpoint services.Endpoint) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	ensureProxyingDisabledForService(endpoint)
	mm.discoveredEndpoints[endpoint.Core().ID] = endpoint

	// If the endpoint has a monitor type specified, then it is expected to
	// have all of its configuration already set in the endpoint and discovery
	// rules will be ignored.
	if endpoint.Core().IsSelfConfigured() {
		if err := mm.monitorSelfConfiguredEndpoint(endpoint); err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"monitorType": endpoint.Core().MonitorType,
				"endpoint":    endpoint,
			}).Error("Could not create monitor for self-configured endpoint")
		}
		return
	}

	mm.findConfigForMonitorAndRun(endpoint)
}

func (mm *MonitorManager) monitorSelfConfiguredEndpoint(endpoint services.Endpoint) error {
	monitorType := endpoint.Core().MonitorType
	conf := &config.MonitorConfig{
		Type: monitorType,
		// This will get overridden by the endpoint configuration if interval
		// was specified
		IntervalSeconds: mm.intervalSeconds,
	}

	monConfig, err := getCustomConfigForMonitor(conf)
	if err != nil {
		return err
	}

	if err = mm.createAndConfigureNewMonitor(monConfig, endpoint); err != nil {
		return err
	}
	return nil
}

func (mm *MonitorManager) findConfigForMonitorAndRun(endpoint services.Endpoint) {
	monitoring := false

	for _, config := range mm.monitorConfigs {
		matched, err := mm.monitorEndpointIfRuleMatches(config, endpoint)
		monitoring = matched || monitoring
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"config":   config,
				"endpoint": endpoint,
			}).Error("Could not monitor new endpoint")
		}
	}

	if !monitoring {
		log.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Debug("Endpoint added that doesn't match any discovery rules")
	}
}

func buildFilterSet(metadata *Metadata, coreConfig *config.MonitorConfig) (*dpfilters.FilterSet, []string, error) {
	oldFilter, err := coreConfig.OldFilterSet()
	if err != nil {
		return nil, nil, err
	}

	newFilter, err := coreConfig.NewFilterSet()
	if err != nil {
		return nil, nil, err
	}

	excludeFilters := []dpfilters.DatapointFilter{oldFilter, newFilter}
	var enabledMetrics []string

	if !metadata.SendAll {
		// Make a copy of extra metrics from config so we don't alter what the user configured.
		extraMetrics := append([]string{}, coreConfig.ExtraMetrics...)

		// Monitors can add additional extra metrics to allow through such as based on config flags.
		if monitorExtra := coreConfig.GetExtraMetrics(); monitorExtra != nil {
			extraMetrics = append(extraMetrics, monitorExtra...)
		}

		includedMetricsFilter, err := newMetricsFilter(metadata, extraMetrics, coreConfig.ExtraGroups)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to construct extraMetrics filter: %s", err)
		}

		// Prepend the included metrics filter.
		excludeFilters = append([]dpfilters.DatapointFilter{dpfilters.Negate(includedMetricsFilter)}, excludeFilters...)
		enabledMetrics = includedMetricsFilter.enabledMetrics()
	} else {
		// Unfortunately can't use a helper because the map value in metadata.Metrics is a non-pointer struct
		// so it's not considered an interface.
		enabledMetrics = make([]string, len(metadata.Metrics))
		i := 0
		for metric := range metadata.Metrics {
			enabledMetrics[i] = metric
			i++
		}
	}

	return &dpfilters.FilterSet{
		ExcludeFilters: excludeFilters,
	}, enabledMetrics, nil
}

// endpoint may be nil for static monitors
func (mm *MonitorManager) createAndConfigureNewMonitor(config config.MonitorCustomConfig, endpoint services.Endpoint) error {
	id := types.MonitorID(mm.idGenerator())
	coreConfig := config.MonitorConfigCore()
	monitorType := coreConfig.Type

	log.WithFields(log.Fields{
		"monitorType":   monitorType,
		"discoveryRule": coreConfig.DiscoveryRule,
		"monitorID":     id,
	}).Info("Creating new monitor")

	instance := newMonitor(config.MonitorConfigCore().Type, id)
	if instance == nil {
		return errors.Errorf("Could not create new monitor of type %s", monitorType)
	}

	metadata, ok := MonitorMetadatas[monitorType]
	if !ok {
		panic(fmt.Sprintf("could not find monitor metadata of type %s", monitorType))
	}

	filterSet, enabledMetrics, err := buildFilterSet(metadata, coreConfig)
	if err != nil {
		return nil
	}

	configHash := config.MonitorConfigCore().Hash()

	output := &monitorOutput{
		monitorType:               coreConfig.Type,
		monitorID:                 id,
		notHostSpecific:           coreConfig.DisableHostDimensions,
		disableEndpointDimensions: coreConfig.DisableEndpointDimensions,
		filterSet:                 filterSet,
		configHash:                configHash,
		endpoint:                  endpoint,
		dpChan:                    mm.DPs,
		eventChan:                 mm.Events,
		dimPropChan:               mm.DimensionProps,
		spanChan:                  mm.TraceSpans,
		extraDims:                 map[string]string{},
	}

	am := &ActiveMonitor{
		id:             id,
		configHash:     configHash,
		instance:       instance,
		endpoint:       endpoint,
		agentMeta:      mm.agentMeta,
		output:         output,
		enabledMetrics: enabledMetrics,
	}

	if err := am.configureMonitor(config); err != nil {
		return err
	}
	mm.activeMonitors = append(mm.activeMonitors, am)

	return nil
}

func (mm *MonitorManager) monitorsForEndpointID(id services.ID) (out []*ActiveMonitor) {
	for i := range mm.activeMonitors {
		if mm.activeMonitors[i].endpointID() == id {
			out = append(out, mm.activeMonitors[i])
		}
	}
	return // Named return value
}

func (mm *MonitorManager) monitorConfigsForType(monitorType string) []*config.MonitorCustomConfig {
	var out []*config.MonitorCustomConfig
	for _, conf := range mm.monitorConfigs {
		if conf.MonitorConfigCore().Type == monitorType {
			out = append(out, &conf)
		}
	}
	return out
}

func (mm *MonitorManager) isServiceMonitored(id services.ID) bool {
	return len(mm.monitorsForEndpointID(id)) > 0
}

// EndpointRemoved should be called by observers when a service endpoint was
// removed.
func (mm *MonitorManager) EndpointRemoved(endpoint services.Endpoint) {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	delete(mm.discoveredEndpoints, endpoint.Core().ID)

	monitors := mm.monitorsForEndpointID(endpoint.Core().ID)
	for _, am := range monitors {
		am.doomed = true
	}
	mm.deleteDoomedMonitors()

	log.WithFields(log.Fields{
		"endpoint": endpoint,
	}).Info("No longer considering endpoint")
}

func (mm *MonitorManager) isEndpointMonitored(endpoint services.Endpoint) bool {
	monitors := mm.monitorsForEndpointID(endpoint.Core().ID)
	return len(monitors) > 0
}

func (mm *MonitorManager) deleteMonitorsByConfigHash(hash uint64) {
	for i := range mm.activeMonitors {
		if mm.activeMonitors[i].configHash == hash {
			log.WithFields(log.Fields{
				"config": mm.activeMonitors[i].config,
			}).Info("Shutting down monitor due to config hash change")
			mm.activeMonitors[i].doomed = true
		}
	}
	mm.deleteDoomedMonitors()
}

func (mm *MonitorManager) deleteDoomedMonitors() {
	newActiveMonitors := []*ActiveMonitor{}

	for i := range mm.activeMonitors {
		am := mm.activeMonitors[i]
		if am.doomed {
			log.WithFields(log.Fields{
				"monitorID":     am.id,
				"monitorType":   am.config.MonitorConfigCore().Type,
				"discoveryRule": am.config.MonitorConfigCore().DiscoveryRule,
			}).Info("Shutting down monitor")

			am.Shutdown()
		} else {
			newActiveMonitors = append(newActiveMonitors, am)
		}
	}

	mm.activeMonitors = newActiveMonitors
}

// Shutdown will shutdown all managed monitors and deinitialize the manager.
func (mm *MonitorManager) Shutdown() {
	mm.lock.Lock()
	defer mm.lock.Unlock()

	for i := range mm.activeMonitors {
		mm.activeMonitors[i].doomed = true
	}
	mm.deleteDoomedMonitors()

	mm.activeMonitors = nil
	mm.discoveredEndpoints = nil
}
