package service

import (
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
)

// queryPerfPaginator allows callers to split up requests for performance data
// for all inventory objects into batches. This is because for large enough
// vSphere deployments, a query for performance data (queryPerf) will fail if
// performance data is requested for all of the inventory objects in one call.
type queryPerfPaginator struct {
	gateway  IGateway
	pageSize int
	log      *log.Entry
}

func (p *queryPerfPaginator) queryPerf(
	objs []*model.InventoryObject,
	maxSample int32,
) (*types.QueryPerfResponse, error) {
	numObjs := len(objs)
	pages := p.numPages(numObjs)
	var metrics []types.BasePerfEntityMetricBase
	for i := 0; i < pages; i++ {
		startIdx := i * p.pageSize
		endIdx := startIdx + p.pageSize
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
	p.log.Debugf("Paginator: %d objects, %d pages", numObjs, pages)
	return &types.QueryPerfResponse{Returnval: metrics}, nil
}

func (p *queryPerfPaginator) numPages(numObjs int) int {
	return (numObjs + p.pageSize - 1) / p.pageSize
}
