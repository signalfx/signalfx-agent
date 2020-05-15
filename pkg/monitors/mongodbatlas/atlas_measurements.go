package mongodbatlas

import (
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"sync"
)

// ProcessMeasurementsGetter for fetching MongoDB process measurements at a given granularity and time period.
type ProcessMeasurementsGetter interface {
	GetProcessMeasurements(*ProcessParams) []*mongodbatlas.Measurements
}

func newOpts(pageNum int) *mongodbatlas.ProcessMeasurementListOptions {
	return &mongodbatlas.ProcessMeasurementListOptions{
		ListOptions: &mongodbatlas.ListOptions{PageNum: pageNum},
		Granularity: "PT1M", // granularity of 1 minute
		Period:      "PT1M", // a period of 1 minute
	}
}

// GetProcessMeasurements fetches metric measurements for a specific process.
func (m *Monitor) GetProcessMeasurements(processParams *ProcessParams) []*mongodbatlas.Measurements {
	var measurements []*mongodbatlas.Measurements
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		measurements = m.getProcessMeasurementsHelper(processParams, 1)
	}()
	for disk := range processParams.disks {
		wg.Add(1)
		go func(disk string) {
			defer wg.Done()
			measurements = append(measurements, m.getDiskProcessMeasurements(processParams.host, processParams.port, disk, 1)...)
		}(disk)
	}
	for db := range processParams.dbs {
		wg.Add(1)
		go func(db string) {
			defer wg.Done()
			measurements = append(measurements, m.getDBProcessMeasurements(processParams.host, processParams.port, db, 1)...)
		}(db)
	}
	wg.Wait()
	return measurements
}

func (m *Monitor) getProcessMeasurementsHelper(processParams *ProcessParams, pageNum int) []*mongodbatlas.Measurements {
	var measurements []*mongodbatlas.Measurements
	processMeasurements, resp, err := m.client.ProcessMeasurements.List(m.ctx, m.projectID, processParams.host, processParams.port, newOpts(pageNum))
	if checkResponseLogErrors(logger, resp, err) {
		return measurements
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return measurements
	}
	if ok {
		measurements = append(measurements, m.getProcessMeasurementsHelper(processParams, nextPage)...)
	}
	for _, m := range processMeasurements.Measurements {
		measurements = append(measurements, m)
	}
	return measurements
}

// getDiskProcessMeasurements fetches disk measurements for a specific process.
func (m *Monitor) getDiskProcessMeasurements(host string, port int, disk string, pageNum int) []*mongodbatlas.Measurements {
	var measurements []*mongodbatlas.Measurements
	diskMeasurements, resp, err := m.client.ProcessDiskMeasurements.List(m.ctx, m.projectID, host, port, disk, newOpts(pageNum))
	if checkResponseLogErrors(logger, resp, err) {
		return measurements
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return measurements
	}
	if ok {
		measurements = append(measurements, m.getDiskProcessMeasurements(host, port, disk, nextPage)...)
	}
	for _, m := range diskMeasurements.Measurements {
		measurements = append(measurements, m)
	}
	return measurements
}

// getDBProcessMeasurements fetches database measurements for a specific process.
func (m *Monitor) getDBProcessMeasurements(host string, port int, db string, pageNum int) []*mongodbatlas.Measurements {
	var measurements []*mongodbatlas.Measurements
	databaseMeasurements, resp, err := m.client.ProcessDatabaseMeasurements.List(m.ctx, m.projectID, host, port, db, newOpts(pageNum))
	if checkResponseLogErrors(logger, resp, err) {
		return measurements
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return measurements
	}
	if ok {
		measurements = append(measurements, m.getDBProcessMeasurements(host, port, db, nextPage)...)
	}
	for _, m := range databaseMeasurements.Measurements {
		measurements = append(measurements, m)
	}
	return measurements
}
