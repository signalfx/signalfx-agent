package measurements

import (
	"sync"
	"sync/atomic"

	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	log "github.com/sirupsen/logrus"
)

// For querying disk partition metric measurements of the MongoDB process hosts.

// disksGetter is for fetching metric measurements of disk partitions in hosts of the MongoDB processes.
type disksGetter struct {
	*config
	disksCache        *atomic.Value
	measurementsCache *atomic.Value
}

// DisksMeasurements are the metric measurements of a particular disk partition in the MongoDB process host.
type DisksMeasurements struct {
	*process
	PartitionName string
	Measurements  []*mongodbatlas.Measurements
}

// measurements gets metric measurements of disk partitions in the hosts of the given MongoDB processes.
func (g *disksGetter) measurements(processes []*process) []*DisksMeasurements {
	disks := getDisks(g, processes)

	var measurements []*DisksMeasurements

	var wg sync.WaitGroup
	for p, partitions := range disks {
		for _, partition := range partitions {
			wg.Add(1)
			go func(p *process, partition string) {
				defer wg.Done()
				measurements = append(measurements, diskMeasurementsHelper(g, p, partition, 1)...)
				if g.enableCache {
					g.measurementsCache.Store(measurements)
				}
			}(p, partition)
		}
	}
	if g.measurementsCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.measurementsCache.Load() != nil && g.enableCache {
		return g.measurementsCache.Load().([]*DisksMeasurements)
	}

	return measurements
}

// diskMeasurementsHelper is a helper function of method measurements.
func diskMeasurementsHelper(g *disksGetter, p *process, diskPartition string, page int) []*DisksMeasurements {
	var measurements []*DisksMeasurements

	measurementsResp, resp, err := g.client.ProcessDiskMeasurements.List(g.ctx, g.projectID, p.Host, p.Port, diskPartition, optionPT1M(page))

	if format, err := errorMsgFormat(err, resp); err != nil {
		log.WithError(err).Errorf(format, "disk measurements", g.projectID, p.Host, p.Port)
		return measurements
	}

	measurements = append(measurements, &DisksMeasurements{process: p, PartitionName: diskPartition, Measurements: measurementsResp.Measurements})

	if ok, next := nextPage(resp); ok {
		measurements = append(measurements, diskMeasurementsHelper(g, p, diskPartition, next)...)
	}

	return measurements
}

// getDisks is a helper function for fetching the names of disk partitions is the hosts of given MongoDB processes.
func getDisks(g *disksGetter, processes []*process) map[*process][]string {
	var disks = make(map[*process][]string)

	var wg sync.WaitGroup

	for _, p := range processes {
		wg.Add(1)
		go func(p *process) {
			defer wg.Done()
			disks[p] = getDisksHelper(g, p, 1)
			if g.enableCache {
				g.disksCache.Store(disks)
			}
		}(p)
	}

	if g.disksCache.Load() == nil || !g.enableCache {
		wg.Wait()
	}

	if g.disksCache.Load() != nil && g.enableCache {
		return g.disksCache.Load().(map[*process][]string)
	}

	return disks
}

// getDisksHelper is a helper function of function getDisks.
func getDisksHelper(g *disksGetter, p *process, page int) []string {
	var disks []string

	disksResp, resp, err := g.client.ProcessDisks.List(g.ctx, g.projectID, p.Host, p.Port, &mongodbatlas.ListOptions{PageNum: page})

	if format, err := errorMsgFormat(err, resp); err != nil {
		log.WithError(err).Errorf(format, "disk partition names", g.projectID, p.Host, p.Port)
		return disks
	}

	for _, r := range disksResp.Results {
		disks = append(disks, r.PartitionName)
	}

	if ok, next := nextPage(resp); ok {
		disks = append(disks, getDisksHelper(g, p, next)...)
	}

	return disks
}
