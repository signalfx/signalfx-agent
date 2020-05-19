package measurements

import (
	"sync"
	"sync/atomic"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	log "github.com/sirupsen/logrus"
)

// databasesGetter is for fetching metric measurements of MongoDB databases.
type databasesGetter struct {
	*config
	databasesCache    *atomic.Value
	measurementsCache *atomic.Value
}

// DatabaseMeasurements are the metric measurements of a MongoDB database.
type DatabaseMeasurements struct {
	*process
	DatabaseName string
	Measurements []*mongodbatlas.Measurements
}

// measurements gets metric measurements of MongoDB databases of the given MongoDB processes.
func (g *databasesGetter) measurements(processes []*process) []*DatabaseMeasurements {
	databases := getDatabases(g, processes)

	var measurements []*DatabaseMeasurements

	var wg sync.WaitGroup
	for p, databaseNames := range databases {
		for _, databaseName := range databaseNames {
			wg.Add(1)
			go func(p *process, databaseName string) {
				defer wg.Done()
				measurements = append(measurements, databaseMeasurementsHelper(g, p, databaseName, 1)...)
				if g.enableCache {
					g.measurementsCache.Store(measurements)
				}
			}(p, databaseName)
		}
	}
	if g.measurementsCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.measurementsCache.Load() != nil && g.enableCache {
		return g.measurementsCache.Load().([]*DatabaseMeasurements)
	}

	return measurements
}

// databaseMeasurementsHelper is a helper function of method measurements.
func databaseMeasurementsHelper(g *databasesGetter, p *process, database string, page int) []*DatabaseMeasurements {
	var measurements []*DatabaseMeasurements

	measurementsResp, resp, err := g.client.ProcessDatabaseMeasurements.List(g.ctx, g.projectID, p.Host, p.Port, database, optionPT1M(page))

	if format, err := formatError(err, resp); err != nil {
		log.WithError(err).Errorf(format, "database measurements", g.projectID, p.Host, p.Port)
		return measurements
	}

	measurements = append(measurements, &DatabaseMeasurements{process: p, DatabaseName: measurementsResp.DatabaseName, Measurements: measurementsResp.Measurements})

	if ok, next := nextPage(resp); ok {
		measurements = append(measurements, databaseMeasurementsHelper(g, p, database, next)...)
	}

	return measurements
}

// getDatabases is a helper function for fetching the names of the MongoDB databases of the given MongoDB processes.
func getDatabases(g *databasesGetter, processes []*process) map[*process][]string {
	var databases = make(map[*process][]string)

	var wg sync.WaitGroup
	for _, p := range processes {
		wg.Add(1)
		go func(p *process) {
			defer wg.Done()
			databases[p] = getDatabasesHelper(g, p, 1)
			if g.enableCache {
				g.databasesCache.Store(databases)
			}
		}(p)
	}
	if g.databasesCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.databasesCache.Load() != nil && g.enableCache {
		return g.databasesCache.Load().(map[*process][]string)
	}

	return databases
}

// getDatabasesHelper is a helper function of function getDatabases.
func getDatabasesHelper(g *databasesGetter, p *process, page int) []string {
	var databases []string

	databasesResp, resp, err := g.client.ProcessDatabases.List(g.ctx, g.projectID, p.Host, p.Port, &mongodbatlas.ListOptions{PageNum: page})

	if format, err := formatError(err, resp); err != nil {
		log.WithError(err).Errorf(format, "database names", g.projectID, p.Host, p.Port)
		return databases
	}

	for _, r := range databasesResp.Results {
		databases = append(databases, r.DatabaseName)
	}

	if ok, next := nextPage(resp); ok {
		databases = append(databases, getDatabasesHelper(g, p, next)...)
	}

	return databases
}
