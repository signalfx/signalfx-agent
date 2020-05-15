package mongodbatlas

import (
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"sync"
)

// MongoDB Atlas exposes metric measurements of MongoDB processes at REST API endpoints. The URLs of these
// measurement endpoints are templatized by host, port, disks and databases parameters.
// See https://docs.atlas.mongodb.com/reference/api/monitoring-and-logs/. This monitor fetches these parameter values
// and use them to construct the endpoint URLs.

// ProcessParams stores fetched measurement endpoint parameter values of a specific process.
type ProcessParams struct {
	host  string              // host name of the process
	port  int                 // the process port on host
	disks map[string]struct{} // disk partition names on the process host
	dbs   map[string]struct{} // database names on the process host
}

type ProcessParamsGetter interface {
	GetProcessParams() []*ProcessParams
}

// GetProcessParams fetches and caches measurement endpoint parameter values of all processes belonging to a project.
func (m *Monitor) GetProcessParams() []*ProcessParams {
	var processParams []*ProcessParams
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processParams = m.getProcessParamsHelper(1)
		if m.enableCache {
			m.processParamsCache.Store(processParams)
		}
	}()
	if m.processParamsCache.Load() == nil || !m.enableCache {
		wg.Wait()
	}
	if m.processParamsCache.Load() != nil && m.enableCache {
		return m.processParamsCache.Load().([]*ProcessParams)
	}
	return processParams
}

var mutex sync.Mutex

func (m *Monitor) getProcessParamsHelper(pageNum int) []*ProcessParams {
	var processParams []*ProcessParams
	processes, resp, err := m.client.Processes.List(m.ctx, m.projectID, &mongodbatlas.ListOptions{PageNum: pageNum})
	if checkResponseLogErrors(logger, resp, err) {
		return processParams
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return processParams
	}
	if ok {
		processParams = append(processParams, m.getProcessParamsHelper(nextPage)...)
	}
	var wg sync.WaitGroup
	for _, process := range processes {
		wg.Add(1)
		go func(process *mongodbatlas.Process) {
			defer wg.Done()
			params := &ProcessParams{host: process.Hostname, port: process.Port, disks: make(map[string]struct{}), dbs: make(map[string]struct{})}
			m.setDisks(params, 1)
			m.setDBs(params, 1)
			mutex.Lock()
			defer mutex.Unlock()
			processParams = append(processParams, params)
		}(process)
	}
	wg.Wait()
	return processParams
}

// setDisks fetches and sets values of the disks field of arg ProcessParams. disks are the disk partition names.
func (m *Monitor) setDisks(processParams *ProcessParams, pageNum int) {
	disks, resp, err := m.client.ProcessDisks.List(m.ctx, m.projectID, processParams.host, processParams.port, &mongodbatlas.ListOptions{PageNum: pageNum})
	if checkResponseLogErrors(logger, resp, err) {
		return
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return
	}
	if ok {
		m.setDisks(processParams, nextPage)
	}
	mutex.Lock()
	defer mutex.Unlock()
	for _, r := range disks.Results {
		processParams.disks[r.PartitionName] = struct{}{}
	}
}

// setDBs fetches and sets values of the dbs field of arg ProcessParams. dbs are the database names.
func (m *Monitor) setDBs(processParams *ProcessParams, pageNum int) {
	dbs, resp, err := m.client.ProcessDatabases.List(m.ctx, m.projectID, processParams.host, processParams.port, &mongodbatlas.ListOptions{PageNum: pageNum})
	if checkResponseLogErrors(logger, resp, err) {
		return
	}
	ok, nextPage, nextPageError := nextPage(resp)
	if logErrors(logger, nextPageError) {
		return
	}
	if ok {
		m.setDBs(processParams, nextPage)
	}
	mutex.Lock()
	defer mutex.Unlock()
	for _, r := range dbs.Results {
		processParams.dbs[r.DatabaseName] = struct{}{}
	}
}
