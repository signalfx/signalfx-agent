package service

import (
	"fmt"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

const fakeMetricKey = 42

var perfMetricSeriesValue = int64(111)

type fakeGateway struct {
	metricIDCounter int32
	sizes           typeCounter
}

type typeCounter struct {
	cluster int
	host    int
	vm      int
}

func getTestingLog() *logrus.Entry {
	return logrus.WithField("monitorType", "vsphere-test")
}

func newFakeGateway() *fakeGateway {
	gateway := &fakeGateway{sizes: typeCounter{
		cluster: 1,
		host:    1,
		vm:      1,
	}}
	return gateway
}

func (g *fakeGateway) retrievePerformanceManager() (*mo.PerformanceManager, error) {
	return &mo.PerformanceManager{
		PerfCounter: []types.PerfCounterInfo{{
			Key:       fakeMetricKey,
			GroupInfo: &types.ElementDescription{Key: "cpu"},
			NameInfo:  &types.ElementDescription{Key: "coreUtilization"},
			StatsType: "delta",
		}},
	}, nil
}

func (g *fakeGateway) retrieveTopLevelFolder() (*mo.Folder, error) {
	return &mo.Folder{
		ChildEntity: []types.ManagedObjectReference{
			{Type: model.DatacenterType, Value: "dc-1"},
		},
	}, nil
}

func (g *fakeGateway) retrieveRefProperties(mor types.ManagedObjectReference, dst interface{}) error {
	switch t := dst.(type) {
	case *mo.Folder:
		t.Self = mor
		t.ChildEntity = g.createRefs(model.ClusterType, "cluster", g.sizes.cluster)
	case *mo.ClusterComputeResource:
		t.Self = mor
		t.Name = "foo cluster"
		t.ComputeResource.Host = g.createRefs(model.HostType, "host", g.sizes.host)
	case *mo.Datacenter:
		t.Self = mor
		t.Name = "foo dc"
	case *mo.HostSystem:
		t.Self = mor
		t.Name = "4.4.4.4"
		t.Config = &types.HostConfigInfo{
			Product: types.AboutInfo{
				OsType: "foo os type",
			},
		}
		t.Vm = g.createRefs(model.VMType, "vm", g.sizes.vm)
	case *mo.VirtualMachine:
		t.Self = mor
		t.Name = "foo vm"
		t.Config = &types.VirtualMachineConfigInfo{
			GuestId: "foo guest id",
		}
		t.Guest = &types.GuestInfo{
			IpAddress:     "1.2.3.4",
			GuestFamily:   "fooFam",
			GuestFullName: "fooFullName",
		}
	default:
		return fmt.Errorf("type not found %v", t)
	}
	return nil
}

//noinspection GoUnusedParameter
func (g *fakeGateway) queryAvailablePerfMetric(ref types.ManagedObjectReference) (*types.QueryAvailablePerfMetricResponse, error) {
	counterID := g.metricIDCounter
	g.metricIDCounter++
	return &types.QueryAvailablePerfMetricResponse{
		Returnval: []types.PerfMetricId{
			{CounterId: counterID, Instance: fmt.Sprintf("instance-%d", counterID)},
		},
	}, nil
}

//noinspection GoUnusedParameter
func (g *fakeGateway) queryPerfProviderSummary(mor types.ManagedObjectReference) (*types.QueryPerfProviderSummaryResponse, error) {
	panic("implement me")
}

//noinspection GoUnusedParameter
func (g *fakeGateway) queryPerf(invObjs []*model.InventoryObject, maxSample int32) (*types.QueryPerfResponse, error) {
	value := perfMetricSeriesValue
	perfMetricSeriesValue++
	m := &types.PerfEntityMetric{
		Value: []types.BasePerfMetricSeries{
			&types.PerfMetricIntSeries{
				PerfMetricSeries: types.PerfMetricSeries{
					Id: types.PerfMetricId{
						CounterId: fakeMetricKey,
					},
				},
				Value: []int64{value},
			},
		},
		SampleInfo: []types.PerfSampleInfo{{Timestamp: time.Time{}}},
		PerfEntityMetricBase: types.PerfEntityMetricBase{
			Entity: types.ManagedObjectReference{Value: "host-0"},
		},
	}
	return &types.QueryPerfResponse{Returnval: []types.BasePerfEntityMetricBase{m}}, nil
}

func (g *fakeGateway) createRefs(key string, prefix string, size int) []types.ManagedObjectReference {
	refs := make([]types.ManagedObjectReference, 0, size)
	for i := 0; i < size; i++ {
		refs = append(refs, types.ManagedObjectReference{Type: key, Value: fmt.Sprintf("%s-%d", prefix, i)})
	}
	return refs
}

func (g *fakeGateway) retrieveCurrentTime() (*time.Time, error) {
	panic("implement me")
}

func (g *fakeGateway) vcenterName() string {
	return "my-vc"
}
