package service

import (
	"testing"

	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/stretchr/testify/require"
)

func TestRetrieveInventory(t *testing.T) {
	gateway := newFakeGateway()
	svc := NewInventorySvc(gateway, getTestingLog())
	inv, _ := svc.RetrieveInventory()
	for _, invObj := range inv.Objects {
		refID := invObj.Ref.Value
		dims := inv.DimensionMap[refID]
		switch invObj.Ref.Type {
		case model.HostType:
			require.Equal(t, "host-0", dims["ref_id"])
			require.Equal(t, model.HostType, dims["object_type"])
			require.Equal(t, "foo host", dims["host_name"])
			require.Equal(t, "foo os type", dims["os_type"])
		case model.VMType:
			require.Equal(t, "vm-0", dims["ref_id"])
			require.Equal(t, model.VMType, dims["object_type"])
			require.Equal(t, "foo vm", dims["vm_name"])
			require.Equal(t, "foo guest id", dims["guest_id"])
		}
	}
}
