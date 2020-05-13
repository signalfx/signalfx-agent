package service

import (
	"testing"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/stretchr/testify/require"
)

func TestRetrieveInventory(t *testing.T) {
	gateway := newFakeGateway(1)
	svc := NewInventorySvc(gateway, testLog)
	inv, _ := svc.RetrieveInventory()
	requireClusterHost(t, inv, 0)
	requireClusterVM(t, inv, 1)
	requireFreeHost(t, inv, 2)
	requireFreeVM(t, inv, 3)
}

func requireClusterHost(t *testing.T, inv *model.Inventory, i int) {
	dims := getDims(inv, i)
	require.Equal(t, "foo cluster", dims["cluster"])
	require.Equal(t, "foo dc", dims["datacenter"])
	require.Equal(t, "4.4.4.4", dims["esx_ip"])
	require.Equal(t, "host-0", dims["ref_id"])
	require.Equal(t, model.HostType, dims["object_type"])
}

func requireClusterVM(t *testing.T, inv *model.Inventory, i int) {
	dims := getDims(inv, i)
	require.Equal(t, "foo cluster", dims["cluster"])
	require.Equal(t, "foo dc", dims["datacenter"])
	require.Equal(t, "4.4.4.4", dims["esx_ip"])
	require.Equal(t, "vm-0", dims["ref_id"])
	require.Equal(t, model.VMType, dims["object_type"])
	require.Equal(t, "foo vm", dims["vm_name"])
	require.Equal(t, "foo guest id", dims["guest_id"])
}

func requireFreeHost(t *testing.T, inv *model.Inventory, i int) {
	dims := getDims(inv, i)
	require.Empty(t, dims["cluster"])
	require.Equal(t, "foo dc", dims["datacenter"])
	require.Equal(t, "4.4.4.4", dims["esx_ip"])
	require.Equal(t, "freehost-1", dims["ref_id"])
	require.Equal(t, model.HostType, dims["object_type"])
}

func requireFreeVM(t *testing.T, inv *model.Inventory, i int) {
	dims := getDims(inv, i)
	require.Empty(t, dims["cluster"])
	require.Equal(t, "foo dc", dims["datacenter"])
	require.Equal(t, "4.4.4.4", dims["esx_ip"])
	require.Equal(t, "vm-1", dims["ref_id"])
	require.Equal(t, model.VMType, dims["object_type"])
	require.Equal(t, "foo vm", dims["vm_name"])
	require.Equal(t, "foo guest id", dims["guest_id"])
}

func getDims(inv *model.Inventory, i int) model.Dimensions {
	return inv.DimensionMap[inv.Objects[i].Ref.Value]
}
