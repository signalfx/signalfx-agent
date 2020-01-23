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
	for _, ref := range topFolder.ChildEntity {
		if ref.Type == model.DatacenterType {
			inv, err = svc.retrieveDatacenter(inv, ref)
			if err != nil {
				return nil, err
			}
		}
	}
	svc.log.Debugf("retrieved inventory: %v", inv.DimensionMap)
	return inv, nil
}

// Retrieves the contents of the passed-in datacenter and puts them in the passed-in Inventory object.
func (svc *InventorySvc) retrieveDatacenter(inv *model.Inventory, dcRef types.ManagedObjectReference) (*model.Inventory, error) {
	var datacenter mo.Datacenter
	err := svc.gateway.retrieveRefProperties(dcRef, &datacenter)
	if err != nil {
		return nil, err
	}
	var dcHostFolder mo.Folder
	err = svc.gateway.retrieveRefProperties(datacenter.HostFolder, &dcHostFolder)
	if err != nil {
		return nil, err
	}
	for _, ref := range dcHostFolder.ChildEntity {
		switch t := ref.Type; t {
		case model.ClusterComputeType:
			var cluster mo.ClusterComputeResource
			err := svc.gateway.retrieveRefProperties(ref, &cluster)
			if err != nil {
				return nil, err
			}
			for _, hostRef := range cluster.ComputeResource.Host {
				err := svc.retrieveHost(inv, hostRef, &datacenter, &cluster)
				if err != nil {
					return nil, err
				}
			}
		case model.ComputeType:
			var computeResource mo.ComputeResource
			err := svc.gateway.retrieveRefProperties(ref, &computeResource)
			if err != nil {
				return nil, err
			}
			for _, hostRef := range computeResource.Host {
				err = svc.retrieveHost(inv, hostRef, &datacenter, nil)
				if err != nil {
					return nil, err
				}
			}
		default:
			svc.log.WithField("ref", ref).Warn("retrieveDatacenter: unknown type")
		}
	}
	return inv, nil
}

func (svc *InventorySvc) retrieveHost(inv *model.Inventory, hostRef types.ManagedObjectReference, datacenter *mo.Datacenter, cluster *mo.ClusterComputeResource) error {
	var host mo.HostSystem
	err := svc.gateway.retrieveRefProperties(hostRef, &host)
	if err != nil {
		return err
	}

	hostDims := map[string]string{
		"esx_ip":     host.Name,
		"datacenter": datacenter.Name,
	}
	if cluster != nil {
		hostDims["cluster"] = cluster.Name
	}
	hostInvObj := model.NewInventoryObject(host.Self, hostDims)
	inv.AddObject(hostInvObj)

	for _, vmRef := range host.Vm {
		var vm mo.VirtualMachine
		err = svc.gateway.retrieveRefProperties(vmRef, &vm)
		if err != nil {
			return err
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
	return nil
}
