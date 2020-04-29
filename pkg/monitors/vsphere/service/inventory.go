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

// use slice semantics to build parent dimensions while traversing the inv tree
type pair [2]string
type pairs []pair

func (svc *InventorySvc) RetrieveInventory() (*model.Inventory, error) {
	inv := model.NewInventory()
	err := svc.followFolder(inv, svc.gateway.topLevelFolderRef(), nil)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (svc *InventorySvc) followFolder(
	inv *model.Inventory,
	parentFolderRef types.ManagedObjectReference,
	dims pairs,
) error {
	var parentFolder mo.Folder
	err := svc.gateway.retrieveRefProperties(parentFolderRef, &parentFolder)
	if err != nil {
		return err
	}
	for _, childRef := range parentFolder.ChildEntity {
		switch childRef.Type {
		case model.FolderType:
			err = svc.followFolder(inv, childRef, dims)
		case model.DatacenterType:
			err = svc.followDatacenter(inv, childRef, dims)
		case model.ClusterComputeType:
			err = svc.followCluster(inv, childRef, dims)
		case model.ComputeType:
			err = svc.followCompute(inv, childRef, dims)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc *InventorySvc) followDatacenter(
	inv *model.Inventory,
	dcRef types.ManagedObjectReference,
	dims pairs,
) error {
	var dc mo.Datacenter
	err := svc.gateway.retrieveRefProperties(dcRef, &dc)
	if err != nil {
		return err
	}
	dims = append(dims, pair{"datacenter", dc.Name})
	// There is also a `dc.VmFolder` but it appears to only receive copies of VMs
	// that live under hosts. Omitting that folder to prevent double counting.
	err = svc.followFolder(inv, dc.HostFolder, dims)
	if err != nil {
		return err
	}
	return nil
}

func (svc *InventorySvc) followCluster(
	inv *model.Inventory,
	clusterRef types.ManagedObjectReference,
	dims pairs,
) error {
	var cluster mo.ClusterComputeResource
	err := svc.gateway.retrieveRefProperties(clusterRef, &cluster)
	if err != nil {
		return err
	}
	dims = append(dims, pair{"cluster", cluster.Name})
	for _, hostRef := range cluster.ComputeResource.Host {
		err = svc.followHost(inv, hostRef, dims)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc *InventorySvc) followCompute(
	inv *model.Inventory,
	computeRef types.ManagedObjectReference,
	dims pairs,
) error {
	var computeResource mo.ComputeResource
	err := svc.gateway.retrieveRefProperties(computeRef, &computeResource)
	if err != nil {
		return err
	}
	for _, hostRef := range computeResource.Host {
		err = svc.followHost(inv, hostRef, dims)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc *InventorySvc) followHost(
	inv *model.Inventory,
	hostRef types.ManagedObjectReference,
	dims pairs,
) error {
	var host mo.HostSystem
	err := svc.gateway.retrieveRefProperties(hostRef, &host)
	if err != nil {
		return err
	}
	dims = append(dims, pair{"esx_ip", host.Name})
	hostDims := map[string]string{}
	amendDims(hostDims, dims)
	hostInvObj := model.NewInventoryObject(host.Self, hostDims)
	inv.AddObject(hostInvObj)
	for _, vmRef := range host.Vm {
		err = svc.followVM(inv, vmRef, dims)
		if err != nil {
			return err
		}
	}
	return nil
}

func (svc *InventorySvc) followVM(
	inv *model.Inventory,
	vmRef types.ManagedObjectReference,
	dims pairs,
) error {
	var vm mo.VirtualMachine
	err := svc.gateway.retrieveRefProperties(vmRef, &vm)
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
	amendDims(vmDims, dims)
	vmInvObj := model.NewInventoryObject(vm.Self, vmDims)
	inv.AddObject(vmInvObj)
	return nil
}

func amendDims(dims map[string]string, pairs pairs) {
	for _, pair := range pairs {
		dims[pair[0]] = pair[1]
	}
}
