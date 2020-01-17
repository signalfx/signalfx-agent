package service

import (
	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Traverses the inventory tree and returns all of the hosts and VMs.
type InventorySvc struct {
	log     *logrus.Entry
	gateway IGateway
}

func NewInventorySvc(gateway IGateway, log *logrus.Entry) *InventorySvc {
	return &InventorySvc{gateway: gateway, log: log}
}

// RetrieveInventory traverses the inventory tree and returns all of the hosts and VMs.
func (svc *InventorySvc) RetrieveInventory() (*model.Inventory, error) {
	topFolder, err := svc.gateway.retrieveTopLevelFolder()
	if err != nil {
		return nil, err
	}
	inv := model.NewInventory()
	inv, err = svc.retrieveFolderContents(topFolder, inv)
	if err != nil {
		return nil, err
	}
	svc.log.Debugf("retrieved inventory: %v", inv.DimensionMap)
	return inv, nil
}

// Retrieves the contents of the passed-in folder and puts it in the passed-in Inventory object.
func (svc *InventorySvc) retrieveFolderContents(folder *mo.Folder, inv *model.Inventory) (*model.Inventory, error) {
	var err error
	for _, ref := range folder.ChildEntity {
		switch t := ref.Type; t {
		case model.DatacenterType:
			inv, err = svc.retrieveDatacenter(ref, inv)
			if err != nil {
				return nil, err
			}
		case model.ClusterType:
			inv, err = svc.retrieveCluster(ref, inv)
			if err != nil {
				return nil, err
			}
		default:
			svc.log.WithField("ref", ref).Warn("retrieveFolderContents: unknown type")
		}
	}
	return inv, nil
}

// Retrieves the contents of the passed-in datacenter and puts them in the passed-in Inventory object.
func (svc *InventorySvc) retrieveDatacenter(ref types.ManagedObjectReference, inv *model.Inventory) (*model.Inventory, error) {
	var datacenter mo.Datacenter
	err := svc.gateway.retrieveRefProperties(ref, &datacenter)
	if err != nil {
		return nil, err
	}

	var dcHostFolder mo.Folder
	err = svc.gateway.retrieveRefProperties(datacenter.HostFolder, &dcHostFolder)
	if err != nil {
		return nil, err
	}
	inv, err = svc.retrieveFolderContents(&dcHostFolder, inv)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// Retrieves the contents of the passed-in cluster and puts it into the passed-in Inventory object.
func (svc *InventorySvc) retrieveCluster(ref types.ManagedObjectReference, inv *model.Inventory) (*model.Inventory, error) {
	var cluster mo.ClusterComputeResource
	err := svc.gateway.retrieveRefProperties(ref, &cluster)
	if err != nil {
		return nil, err
	}

	for _, hostRef := range cluster.ComputeResource.Host {
		var host mo.HostSystem
		err = svc.gateway.retrieveRefProperties(hostRef, &host)
		if err != nil {
			return nil, err
		}

		hostDims := map[string]string{
			"esx_ip": host.Name,
		}
		hostInvObj := model.NewInventoryObject(host.Self, hostDims)
		inv.AddObject(hostInvObj)

		for _, vmRef := range host.Vm {
			var vm mo.VirtualMachine
			err = svc.gateway.retrieveRefProperties(vmRef, &vm)
			if err != nil {
				return nil, err
			}

			vmDims := map[string]string{
				"vm_name":        vm.Name,           // e.g. "MyDebian10Host"
				"guest_id":       vm.Config.GuestId, // e.g. "debian10_64Guest"
				"vm_ip":          vm.Guest.IpAddress,
				"guest_family":   vm.Guest.GuestFamily,   // e.g. "linuxGuest"
				"guest_fullname": vm.Guest.GuestFullName, // e.g. "Other 4.x or later Linux (64-bit)"
			}
			updateMap(vmDims, hostDims)
			vmInvObj := model.NewInventoryObject(vm.Self, vmDims)
			inv.AddObject(vmInvObj)
		}
	}
	return inv, nil
}
