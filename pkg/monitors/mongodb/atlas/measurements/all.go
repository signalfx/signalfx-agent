package measurements

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

// Measurements is a composite of all (process, disk and database) metric measurements of MongoDB processes.
type Measurements struct {
	Processes []*ProcessMeasurements
	Disks     []*DisksMeasurements
	Databases []*DatabaseMeasurements
}

// Getter is for getting all metric measurements of MongoDB processes.
type Getter interface {
	GetAll() Measurements
}

// getter implements Getter and gets all metric measurements of all MongoDB processes of the configured project ID.
type getter struct {
	*config
	processes *processesGetter
	disks     *disksGetter
	databases *databasesGetter
}

// NewGetter returns a value that implements Getter.
func NewGetter(ctx context.Context, client *mongodbatlas.Client, projectID string, enableCache bool) Getter {
	conf := &config{
		ctx:         ctx,
		client:      client,
		projectID:   projectID,
		enableCache: enableCache,
	}

	return &getter{
		config:    conf,
		processes: &processesGetter{config: conf, processesCache: new(atomic.Value), measurementsCache: new(atomic.Value)},
		disks:     &disksGetter{config: conf, disksCache: new(atomic.Value), measurementsCache: new(atomic.Value)},
		databases: &databasesGetter{config: conf, databasesCache: new(atomic.Value), measurementsCache: new(atomic.Value)},
	}

}

// GetAll gets all metric measurements of all MongoDB processes of the configured project ID.
func (g *getter) GetAll() Measurements {
	var measurements Measurements

	processes := getProcesses(g.processes)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		measurements.Processes = g.processes.measurements(processes)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		measurements.Disks = g.disks.measurements(processes)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		measurements.Databases = g.databases.measurements(processes)
	}()

	wg.Wait()

	return measurements
}
