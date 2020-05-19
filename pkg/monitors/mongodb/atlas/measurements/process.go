package measurements

import (
	"fmt"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// processesGetter is for getting metric measurements of MongoDB processes.
type processesGetter struct {
	*config
	processesCache    *atomic.Value
	measurementsCache *atomic.Value
}

// ProcessMeasurements are the metric measurements of a particular MongoDB process.
type ProcessMeasurements struct {
	*process
	Measurements []*mongodbatlas.Measurements
}

// process is the MongoDB process identified by the host and port on which the process is running.
type process struct {
	Host string // The name of the host in which the MongoDB process is running
	Port int    // The port number on which the MongoDB process is running
}

// measurements gets metric measurements of the given MongoDB processes.
func (g *processesGetter) measurements(processes []*process) []*ProcessMeasurements {
	var measurements []*ProcessMeasurements

	var wg sync.WaitGroup
	for _, p := range processes {
		wg.Add(1)
		go func(p *process) {
			defer wg.Done()
			measurements = append(measurements, processMeasurementsHelper(g, p, 1)...)
			if g.enableCache {
				g.measurementsCache.Store(measurements)
			}
		}(p)
	}
	if g.measurementsCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.measurementsCache.Load() != nil && g.enableCache {
		return g.measurementsCache.Load().([]*ProcessMeasurements)
	}

	return measurements
}

// processMeasurementsHelper is a helper function of method measurements.
func processMeasurementsHelper(g *processesGetter, p *process, pageNum int) []*ProcessMeasurements {
	var measurements []*ProcessMeasurements

	processMeasurements, resp, err := g.client.ProcessMeasurements.List(g.ctx, g.projectID, p.Host, p.Port, optionPT1M(pageNum))

	if format, err := errorMsgFormat(err, resp); err != nil {
		log.WithError(err).Errorf(format, "process measurements", g.projectID, p.Host, p.Port)
		return measurements
	}

	measurements = append(measurements, &ProcessMeasurements{process: p, Measurements: processMeasurements.Measurements})

	if ok, next := nextPage(resp); ok {
		measurements = append(measurements, processMeasurementsHelper(g, p, next)...)
	}

	return measurements
}

// get gets all MongoDB processes in the configured project ID.
func getProcesses(g *processesGetter) []*process {
	var processes []*process

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		processes = getProcessesHelper(g, 1)
		if g.enableCache {
			g.processesCache.Store(processes)
		}
	}()

	if g.processesCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.processesCache.Load() != nil && g.enableCache {
		return g.processesCache.Load().([]*process)
	}

	return processes
}

// getProcessesHelper is a helper function for method get.
func getProcessesHelper(g *processesGetter, pageNum int) []*process {
	var processes []*process

	processesResp, resp, err := g.client.Processes.List(g.ctx, g.projectID, &mongodbatlas.ListOptions{PageNum: pageNum})

	if err != nil {
		log.WithError(err).Error(fmt.Sprintf("the request for getting processes failed (Atlas project: %s)", g.projectID))
		return processes
	}

	if resp == nil {
		log.Errorf("the response for getting processes returned empty (Atlas project: %s)", g.projectID)
		return processes
	}

	if err := mongodbatlas.CheckResponse(resp.Response); err != nil {
		log.WithError(err).Error(fmt.Sprintf("the response for getting processes returned an error (Atlas project: %s)", g.projectID))
		return processes
	}

	for _, p := range processesResp {
		processes = append(processes, &process{Host: p.Hostname, Port: p.Port})
	}

	if ok, next := nextPage(resp); ok {
		processes = append(processes, getProcessesHelper(g, next)...)
	}

	return processes
}
