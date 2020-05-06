package service

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
)

// queryPerfPaginator allows callers to split up requests for performance data
// for all inventory objects into batches. This is because for large enough
// vSphere deployments, a query for performance data (queryPerf) will fail if
// performance data is requested for all of the inventory objects in one call.
type queryPerfPaginator struct {
	gateway IGateway
	// The max number of inventory objects requested at a time.
	// A pagesize of 0 will result in no pagination (one call for all inv objects)
	pageSize int
	log      *log.Entry
}

func (p *queryPerfPaginator) paginate(
	objs []*model.InventoryObject,
	maxSample int32,
) (*types.QueryPerfResponse, error) {
	numObjs := len(objs)
	pages := p.numPages(numObjs)
	pageSize := p.pageSize
	if pageSize == 0 {
		pageSize = numObjs
	}
	start := time.Now()
	var metrics []types.BasePerfEntityMetricBase
	for i := 0; i < pages; i++ {
		startIdx := i * pageSize
		endIdx := startIdx + pageSize
		if endIdx > numObjs {
			endIdx = numObjs
		}
		slice := objs[startIdx:endIdx]
		perf, err := p.gateway.queryPerf(slice, maxSample)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, perf.Returnval...)
	}
	end := time.Now()
	duration := end.Sub(start)
	p.log.Debugf("Paginator: %d pages took %v", pages, duration)
	return &types.QueryPerfResponse{Returnval: metrics}, nil
}

func (p *queryPerfPaginator) numPages(n int) int {
	// no pagination if page size is zero
	if p.pageSize == 0 {
		return 1
	}
	return (n + p.pageSize - 1) / p.pageSize
}
