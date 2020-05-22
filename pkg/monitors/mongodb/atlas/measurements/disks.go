package measurements

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	log "github.com/sirupsen/logrus"
)

// DisksMeasurements are the metric measurements of a particular disk partition in a MongoDB process host.
type DisksMeasurements map[Process]struct {
	DiskName     string
	Measurements []*mongodbatlas.Measurements
}

// DisksGetter is for fetching metric measurements of disk partitions in the MongoDB processes hosts.
type DisksGetter interface {
	GetMeasurements(ctx context.Context, timeout time.Duration, processes []Process) DisksMeasurements
}

// disksGetter implements DisksGetter
type disksGetter struct {
	projectID         string
	client            *mongodbatlas.Client
	enableCache       bool
	mutex             *sync.Mutex
	measurementsCache *atomic.Value
	disksCache        *atomic.Value
}

// NewProcessesGetter returns a new ProcessesGetter.
func NewDisksGetter(projectID string, client *mongodbatlas.Client, enableCache bool) DisksGetter {
	return &disksGetter{
		projectID:         projectID,
		client:            client,
		enableCache:       enableCache,
		mutex:             new(sync.Mutex),
		measurementsCache: new(atomic.Value),
		disksCache:        new(atomic.Value),
	}
}

// GetMeasurements gets metric measurements of disk partitions in the hosts of the given MongoDB processes.
func (getter *disksGetter) GetMeasurements(ctx context.Context, timeout time.Duration, processes []Process) DisksMeasurements {
	var measurements = make(DisksMeasurements)

	disks := getter.getDisks(ctx, timeout, processes)

	var wg1 sync.WaitGroup

	wg1.Add(1)

	go func() {
		defer wg1.Done()

		var wg2 sync.WaitGroup
		for process, diskNames := range disks {
			for _, diskName := range diskNames {
				wg2.Add(1)

				go func(process Process, diskName string) {
					defer wg2.Done()

					var ctx, cancel = context.WithTimeout(ctx, timeout)
					defer cancel()

					getter.setMeasurements(ctx, measurements, process, diskName, 1)
				}(process, diskName)
			}
		}
		wg2.Wait()

		if getter.enableCache {
			getter.measurementsCache.Store(measurements)
		}
	}()

	if getter.measurementsCache.Load() != nil && getter.enableCache {
		return getter.measurementsCache.Load().(DisksMeasurements)
	}

	wg1.Wait()

	return measurements
}

// setMeasurements is a helper function of method GetMeasurements.
func (getter *disksGetter) setMeasurements(ctx context.Context, disksMeasurements DisksMeasurements, process Process, diskName string, page int) {
	list, resp, err := getter.client.ProcessDiskMeasurements.List(ctx, getter.projectID, process.Host, process.Port, diskName, optionPT1M(page))

	if msg, err := errorMsg(err, resp); err != nil {
		log.WithError(err).Errorf(msg, "disk measurements", getter.projectID, process.Host, process.Port)
		return
	}

	if ok, next := nextPage(resp); ok {
		getter.setMeasurements(ctx, disksMeasurements, process, diskName, next)
	}

	getter.mutex.Lock()
	defer getter.mutex.Unlock()

	disksMeasurements[process] = struct {
		DiskName     string
		Measurements []*mongodbatlas.Measurements
	}{DiskName: diskName, Measurements: list.Measurements}
}

// getDisks is a helper function for fetching the names of disk partitions is the hosts of given MongoDB processes.
func (getter *disksGetter) getDisks(ctx context.Context, timeout time.Duration, processes []Process) map[Process][]string {
	var disks = make(map[Process][]string)

	var wg1 sync.WaitGroup

	wg1.Add(1)

	go func() {
		defer wg1.Done()

		var wg2 sync.WaitGroup
		for _, process := range processes {
			wg2.Add(1)

			go func(process Process) {
				defer wg2.Done()

				var ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()

				diskNames := getter.getDiskNames(ctx, process, 1)

				getter.mutex.Lock()
				defer getter.mutex.Unlock()
				disks[process] = diskNames
			}(process)
		}
		wg2.Wait()

		if getter.enableCache {
			getter.disksCache.Store(disks)
		}
	}()

	if getter.disksCache.Load() != nil && getter.enableCache {
		return getter.disksCache.Load().(map[Process][]string)
	}

	wg1.Wait()

	return disks
}

// getDiskNames is a helper function of function getDisks.
func (getter *disksGetter) getDiskNames(ctx context.Context, process Process, page int) (names []string) {
	list, resp, err := getter.client.ProcessDisks.List(ctx, getter.projectID, process.Host, process.Port, &mongodbatlas.ListOptions{PageNum: page})

	if msg, err := errorMsg(err, resp); err != nil {
		log.WithError(err).Errorf(msg, "disk partition names", getter.projectID, process.Host, process.Port)
		return names
	}

	if ok, next := nextPage(resp); ok {
		names = append(names, getter.getDiskNames(ctx, process, next)...)
	}

	for _, r := range list.Results {
		names = append(names, r.PartitionName)
	}

	return names
}
