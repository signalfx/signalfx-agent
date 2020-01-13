package service

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// A thin wrapper around the vmomi SDK so that callers don't have to use it directly.
type IGateway interface {
	retrievePerformanceManager() (*mo.PerformanceManager, error)
	retrieveTopLevelFolder() (*mo.Folder, error)
	retrieveRefProperties(mor types.ManagedObjectReference, dst interface{}) error
	queryAvailablePerfMetric(ref types.ManagedObjectReference) (*types.QueryAvailablePerfMetricResponse, error)
	queryPerfProviderSummary(mor types.ManagedObjectReference) (*types.QueryPerfProviderSummaryResponse, error)
	queryPerf(invObjs []*model.InventoryObject, maxSample int32) (*types.QueryPerfResponse, error)
	retrieveCurrentTime() (*time.Time, error)
	vcenterName() string
}

type Gateway struct {
	ctx    context.Context
	client *govmomi.Client
	vcName string
}

func NewGateway(ctx context.Context, client *govmomi.Client) *Gateway {
	return &Gateway{
		ctx:    ctx,
		client: client,
		vcName: client.Client.URL().Host,
	}
}

func (g *Gateway) retrievePerformanceManager() (*mo.PerformanceManager, error) {
	var pm mo.PerformanceManager
	err := mo.RetrieveProperties(
		g.ctx,
		g.client,
		g.client.ServiceContent.PropertyCollector,
		*g.client.Client.ServiceContent.PerfManager,
		&pm,
	)
	return &pm, err
}

func (g *Gateway) retrieveTopLevelFolder() (*mo.Folder, error) {
	var folder mo.Folder
	err := mo.RetrieveProperties(
		g.ctx,
		g.client,
		g.client.ServiceContent.PropertyCollector,
		g.client.ServiceContent.RootFolder,
		&folder,
	)
	return &folder, err
}

func (g *Gateway) retrieveRefProperties(ref types.ManagedObjectReference, dst interface{}) error {
	return mo.RetrieveProperties(
		g.ctx,
		g.client,
		g.client.ServiceContent.PropertyCollector,
		ref,
		dst,
	)
}

func (g *Gateway) queryAvailablePerfMetric(ref types.ManagedObjectReference) (*types.QueryAvailablePerfMetricResponse, error) {
	req := types.QueryAvailablePerfMetric{
		This:       *g.client.Client.ServiceContent.PerfManager,
		Entity:     ref,
		IntervalId: model.RealtimeMetricsInterval,
	}
	return methods.QueryAvailablePerfMetric(g.ctx, g.client.Client, &req)
}

func (g *Gateway) queryPerfProviderSummary(ref types.ManagedObjectReference) (*types.QueryPerfProviderSummaryResponse, error) {
	req := types.QueryPerfProviderSummary{
		This:   *g.client.Client.ServiceContent.PerfManager,
		Entity: ref,
	}
	return methods.QueryPerfProviderSummary(g.ctx, g.client.Client, &req)
}

func (g *Gateway) queryPerf(invObjs []*model.InventoryObject, maxSample int32) (*types.QueryPerfResponse, error) {
	specs := make([]types.PerfQuerySpec, 0, len(invObjs))
	for _, invObj := range invObjs {
		specs = append(specs, types.PerfQuerySpec{
			Entity:     invObj.Ref,
			MaxSample:  maxSample,
			IntervalId: model.RealtimeMetricsInterval,
			MetricId:   invObj.MetricIds,
		})
	}
	queryPerf := types.QueryPerf{
		This:      *g.client.Client.ServiceContent.PerfManager,
		QuerySpec: specs,
	}
	return methods.QueryPerf(g.ctx, g.client.Client, &queryPerf)
}

func (g *Gateway) retrieveCurrentTime() (*time.Time, error) {
	return methods.GetCurrentTime(g.ctx, g.client)
}

func (g *Gateway) vcenterName() string {
	return g.vcName
}
