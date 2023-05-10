/*
Copyright (c) Intel Corporation.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog"

	opiapiCommon "github.com/opiproject/opi-api/common/v1/gen/go"
	opiapiStorage "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

const (
	opiNvmfRemoteControllerHostnqnPref = "nqn.2023-04.io.spdk.csi:remote.controller:uuid:"
	opiNVMeSubsystemNqnPref            = "nqn.2016-06.io.spdk.csi:subsystem:uuid:"
	nvmfRemoteControllerPrefix         = "spckcsi-"
	nvmeSubSystemPrefix                = "spdkcsi-"
	nvmeControllerPrefix               = "spdkcsi-"
	nvmeNamespacePrefix                = "spdcksi-"
	virtioBlkPrefix                    = "virtioBlk."

	// volumePrefix = "//storage.opiproject.org/volumes/"
)

type opiCommon struct {
	opiClient                  *grpc.ClientConn
	nvmfRemoteControllerClient opiapiStorage.NVMfRemoteControllerServiceClient
	volumeContext              map[string]string
	timeout                    time.Duration
	kvmPciBridges              int
	devicePath                 string
	nvmfRemoteControllerName   string
}

type opiInitiatorNvme struct {
	*opiCommon
	frontendNvmeClient opiapiStorage.FrontendNvmeServiceClient
	lock               sync.Mutex
	subsystemName      string
	namespaceName      string
	nvmeControllerName string
}

type opiInitiatorVirtioBlk struct {
	*opiCommon
	frontendVirtioBlkClient opiapiStorage.FrontendVirtioBlkServiceClient
	virtioBlkName           string
}

func NewSpdkCsiOpiInitiator(volumeContext map[string]string, xpuClient *grpc.ClientConn, xpuTargetType string, kvmPciBridges int) (SpdkCsiInitiator, error) {
	iOpiCommon := &opiCommon{
		opiClient:                  xpuClient,
		volumeContext:              volumeContext,
		timeout:                    60 * time.Second,
		kvmPciBridges:              kvmPciBridges,
		nvmfRemoteControllerClient: opiapiStorage.NewNVMfRemoteControllerServiceClient(xpuClient),
	}

	switch xpuTargetType {
	case "xpu-opi-virtioblk":
		return &opiInitiatorVirtioBlk{
			opiCommon:               iOpiCommon,
			frontendVirtioBlkClient: opiapiStorage.NewFrontendVirtioBlkServiceClient(xpuClient),
		}, nil
	case "xpu-opi-nvme":
		return &opiInitiatorNvme{
			opiCommon:          iOpiCommon,
			frontendNvmeClient: opiapiStorage.NewFrontendNvmeServiceClient(xpuClient),
		}, nil
	default:
		return nil, fmt.Errorf("unknown xPU targetType: %s", xpuTargetType)
	}
}

func (opi *opiCommon) ctxTimeout() (context.Context, context.CancelFunc) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), opi.timeout)
	return ctxTimeout, cancel
}

// Connect to remote controller, which is needed for by OPI VirtioBlk and Nvme
func (opi *opiCommon) createNvmfRemoteController() error {
	ctxTimeout, cancel := opi.ctxTimeout()
	defer cancel()

	nvmfRemoteControllerID := nvmfRemoteControllerPrefix + opi.volumeContext["model"]

	targetSvcPort, err := strconv.ParseInt(opi.volumeContext["targetPort"], 10, 64)
	if err != nil {
		return fmt.Errorf("failed to create remote NVMf controller for '%s': invalid targetPort '%s': %w",
			nvmfRemoteControllerID, opi.volumeContext["targetPort"], err)
	}
	createReq := &opiapiStorage.CreateNVMfRemoteControllerRequest{
		NvMfRemoteController: &opiapiStorage.NVMfRemoteController{
			Trtype:  opiapiStorage.NvmeTransportType_NVME_TRANSPORT_TCP,
			Adrfam:  opiapiStorage.NvmeAddressFamily_NVMF_ADRFAM_IPV4,
			Traddr:  opi.volumeContext["targetAddr"],
			Trsvcid: targetSvcPort,
			Subnqn:  opi.volumeContext["nqn"],
			Hostnqn: opiNvmfRemoteControllerHostnqnPref + opi.volumeContext["model"],
		},
	}

	klog.Infof("OPI.CreateNVMfRemoteControllerRequest(...) => %s", createReq)
	var createResp *opiapiStorage.NVMfRemoteController
	createResp, err = opi.nvmfRemoteControllerClient.CreateNVMfRemoteController(ctxTimeout, createReq)
	if err != nil {
		return fmt.Errorf("failed to create remote NVMf controller for '%s': %w", nvmfRemoteControllerID, err)
	}
	klog.Info("OPI.CreateNVMfRemoteController(...) => ", createResp)
	opi.nvmfRemoteControllerName = createResp.Name
	return nil
}

// Disconnect from remote controller, which is needed by both OPI VirtioBlk and Nvme
func (opi *opiCommon) deleteNvmfRemoteController() error {
	if opi.nvmfRemoteControllerName == "" {
		return nil
	}
	ctxTimeout, cancel := opi.ctxTimeout()
	defer cancel()

	// DeleteNVMfRemoteController, with "AllowMissing: true", deleting operation will always succeed even the resource is not found
	deleteReq := &opiapiStorage.DeleteNVMfRemoteControllerRequest{
		Name:         opi.nvmfRemoteControllerName,
		AllowMissing: true,
	}
	klog.Infof("OPI.DeleteNVMfRemoteControllerRequest(...) => %s", deleteReq)
	if _, err := opi.nvmfRemoteControllerClient.DeleteNVMfRemoteController(ctxTimeout, deleteReq); err != nil {
		klog.Infof("Error on deleting remote NVMf controller '%s': %v", opi.nvmfRemoteControllerName, err)
		return err
	}
	klog.Info("OPI.DeleteNVMfRemoteController successfully")
	opi.nvmfRemoteControllerName = ""

	return nil
}

// Create the subsystem
func (i *opiInitiatorNvme) createNVMeSubsystem() error {
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	nvmeSubsystemID := nvmeSubSystemPrefix + i.volumeContext["model"]

	// Create the subsystem if it does not exist
	createReq := &opiapiStorage.CreateNvmeSubsystemRequest{
		NvmeSubsystemId: nvmeSubsystemID,
		NvmeSubsystem: &opiapiStorage.NvmeSubsystem{
			Spec: &opiapiStorage.NvmeSubsystemSpec{
				Nqn: opiNVMeSubsystemNqnPref + i.volumeContext["model"],
			},
		},
	}
	klog.Infof("OPI.CreateNVMeSubsystemRequest(...) => %s", createReq)
	createResp, err := i.frontendNvmeClient.CreateNvmeSubsystem(ctxTimeout, createReq)
	if err != nil {
		return fmt.Errorf("failed to create NVMe subsystem for '%s': %w", nvmeSubsystemID, err)
	}
	klog.Info("OPI.CreateNVMeSubsystem(...) => ", createResp)
	i.subsystemName = createResp.Name

	return nil
}

// deleteNVMeSubsystem
func (i *opiInitiatorNvme) deleteNVMeSubsystem() error {
	if i.subsystemName == "" {
		return nil
	}
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	// Delete the subsystem, with "AllowMissing: true", deleting operation will always succeed even the resource is not found
	deleteReq := &opiapiStorage.DeleteNvmeSubsystemRequest{
		Name:         i.subsystemName,
		AllowMissing: true,
	}
	klog.Infof("OPI.DeleteNVMeSubsystemRequest(...) => %s", deleteReq)
	if _, err := i.frontendNvmeClient.DeleteNvmeSubsystem(ctxTimeout, deleteReq); err != nil {
		return fmt.Errorf("failed to delete NVMe subsystem '%s': %w", i.subsystemName, err)
	}
	klog.Info("OPI.DeleteNVMeSubsystem successfully")
	i.subsystemName = ""

	return nil
}

// Create a controller with vfiouser transport information for Nvme
func (i *opiInitiatorNvme) createNVMeController() (bdf string, err error) {
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	devicePath, err := CheckIfNvmeDeviceExists(i.volumeContext["model"], nil)
	if devicePath != "" {
		klog.Infof("Found existing device for '%s': %v", i.volumeContext["mode"], devicePath)
		return devicePath, nil
	}
	if !os.IsNotExist(err) {
		klog.Errorf("failed to detect existing nvme device for '%s'", i.volumeContext["model"])
	}
	pf, vf, err := GetNvmeAvailableFunction(i.kvmPciBridges)
	if err != nil {
		return "", fmt.Errorf("failed to detect free NVMe virtual function: %w", err)
	}
	physID := pf*32 + vf
	nvmeControllerID := nvmeControllerPrefix + strconv.Itoa(int(physID))
	// Create the controller with vfiouser transport information if it does not exist
	createReq := &opiapiStorage.CreateNvmeControllerRequest{
		NvmeController: &opiapiStorage.NvmeController{
			Spec: &opiapiStorage.NvmeControllerSpec{
				SubsystemId: &opiapiCommon.ObjectKey{
					Value: i.subsystemName,
				},
				PcieId: &opiapiStorage.PciEndpoint{
					PhysicalFunction: int32(physID),
				},
			},
		},
		NvmeControllerId: nvmeControllerID,
	}
	klog.Infof("OPI.CreateNVMeControllerRequest(...) => %s", createReq)

	createResp, err := i.frontendNvmeClient.CreateNvmeController(ctxTimeout, createReq)
	if err != nil {
		klog.Errorf("OPI.CreateNVMeController with pfId (%d) error: %s", physID, err)
		return "", fmt.Errorf("failed to create NVMe controller: %w", err)
	}
	klog.Info("OPI.CreateNVMeController(...) with pfId => ", physID, createResp)
	i.nvmeControllerName = createResp.Name

	return fmt.Sprintf("0000:%02x:%02x.0", pf+1, vf), nil
}

// Delete the controller with vfiouser transport information for Nvme
func (i *opiInitiatorNvme) deleteNVMeController() (err error) {
	if i.nvmeControllerName == "" {
		return nil
	}
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	// Delete the controller with vfiouser transport information, with "AllowMissing: true", deleting operation will always succeed even the resource is not found
	deleteControllerReq := &opiapiStorage.DeleteNvmeControllerRequest{
		Name:         i.nvmeControllerName,
		AllowMissing: true,
	}
	klog.Infof("OPI.DeleteNVMeControllerRequest(...) => %s", deleteControllerReq)
	if _, err = i.frontendNvmeClient.DeleteNvmeController(ctxTimeout, deleteControllerReq); err != nil {
		klog.Errorf("OPI.Nvme DeleteNVMeController error: %s", err)
		return err
	}
	klog.Info("OPI.DeleteNVMeController successfully")
	i.nvmeControllerName = ""

	return nil
}

// Get Bdev for the volume and add a new namespace to the subsystem with that bdev for Nvme
func (i *opiInitiatorNvme) createNVMeNamespace() error {
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	nvmeNamespaceID := nvmeNamespacePrefix + i.volumeContext["model"]
	createReq := &opiapiStorage.CreateNvmeNamespaceRequest{
		NvmeNamespace: &opiapiStorage.NvmeNamespace{
			Spec: &opiapiStorage.NvmeNamespaceSpec{
				SubsystemId: &opiapiCommon.ObjectKey{
					Value: i.subsystemName,
				},
				VolumeId: &opiapiCommon.ObjectKey{
					Value: i.volumeContext["model"],
				},
			},
		},
		NvmeNamespaceId: nvmeNamespaceID,
	}
	klog.Infof("OPI.CreateNVMeNamespaceRequest(...) => %s", createReq)
	createResp, err := i.frontendNvmeClient.CreateNvmeNamespace(ctxTimeout, createReq)
	if err != nil {
		klog.Infof("Failed to create nvme namespace '%s': %v", nvmeNamespaceID, err)
		return err
	}
	klog.Info("OPI.CreateNVMeNamespace(...) => ", createResp)
	i.namespaceName = createResp.Name
	return nil
}

// Delete the namespace from the subsystem with the bdev for Nvme
func (i *opiInitiatorNvme) deleteNVMeNamespace() error {
	if i.namespaceName == "" {
		return nil
	}
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	// Delete namespace, with "AllowMissing: true", deleting operation will always succeed even the resource is not found
	deleteNvmeNamespaceReq := &opiapiStorage.DeleteNvmeNamespaceRequest{
		Name:         i.namespaceName,
		AllowMissing: true,
	}
	klog.Infof("OPI.DeleteNVMeNamespaceRequest(...) => %s", deleteNvmeNamespaceReq)

	if _, err := i.frontendNvmeClient.DeleteNvmeNamespace(ctxTimeout, deleteNvmeNamespaceReq); err != nil {
		klog.Errorf("OPI.Nvme DeleteNVMeNamespace error: %s", err)
		return err
	}
	klog.Info("OPI.DeleteNVMeNamespace successfully")
	i.namespaceName = ""

	return nil
}

// cleanup for OPI Nvme
func (i *opiInitiatorNvme) cleanup() {
	// All the deleting operations have "AllowMissing: true" in the request, they will always succeed even the resources are not found
	// So te cleanup contains all the resources deleting operations
	if err := i.deleteNVMeSubsystem(); err != nil {
		klog.Info("OPI.Nvme workflow failed, call Delete* to clean up err: ", err)
	}

	if err := i.deleteNVMeController(); err != nil {
		klog.Info("OPI.Nvme workflow failed, call Delete* to clean up err:", err)
	}

	if err := i.deleteNvmfRemoteController(); err != nil {
		klog.Info("OPI.Nvme workflow failed, call Delete* to clean up err:", err)
	}

	if err := i.deleteNVMeNamespace(); err != nil {
		klog.Info("OPI.Nvme workflow failed, call Delete* to clean up err:", err)
	}
}

// For OPI Nvme Connect(), four steps will be included.
// The first step is Create a new subsystem, the nqn (nqn.2016-06.io.spdk.csi:subsystem:uuid:VolumeId) will be set in the CreateNVMeSubsystemRequest.
// After a successful CreateNVMeSubsystem, a nvmf subsystem with the nqn will be created in the xPU node.
// The second step is create a controller with vfiouser transport information, we are using KVM case now,
// and the only information needed in the CreateNVMeControllerRequest is pfId
// which should be from 0 to the sum of buses-counts (namely 64 in our case). After a successful CreateNVMeController, the "listen_addresses" field in the nvmf subsystem
// created in the first step will be filled in with VFIOUSER related information,
// including transport (VFIOUSER), trtype (VFIOUSER), adrfam (IPv4) and traddr (/var/tmp/controller$pfId).
// The third step is to connect to the remote controller, this step is used to connect to the storage node.
// The last step is to get Bdev for the volume and add a new namespace to the subsystem with that bdev. After this step, the Nvme block device will appear.
// If any step above fails, call cleanup operation to clean the resources.
func (i *opiInitiatorNvme) Connect() (string, error) {
	failed := true
	// step 1: create a subsystem
	if err := i.createNVMeSubsystem(); err != nil {
		return "", err
	}
	defer func() {
		if failed {
			klog.Info("Cleaning up incomplete NVMe creation...")
			i.cleanup()
		}
	}()
	// step 2: create a controller with vfiouser transport information
	bdf, err := i.createNVMeController()
	if err != nil {
		return "", err
	}
	// step 3: connect to remote controller
	if err = i.createNvmfRemoteController(); err != nil {
		return "", err
	}
	// step 4: get Bdev for the volume and add a new namespace to the subsystem with that bdev
	if err = i.createNVMeNamespace(); err != nil {
		return "", err
	}
	// wait for the block device ready
	devicePath, err := GetNvmeDeviceName(i.volumeContext["model"], bdf)
	if err != nil {
		return "", err
	}

	failed = false
	i.devicePath = devicePath

	return devicePath, nil
}

// For OPI Nvme Disconnect(), three steps will be included, namely DeleteNVMfRemoteController, DeleteNVMeController and DeleteNVMeSubsystem.
// DeleteNVMeNamespace is skipped cause when deleting subsystem, namespace will be deleted automatically
func (i *opiInitiatorNvme) Disconnect() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	// step 1: deleteNVMfRemoteController
	if err := i.deleteNvmfRemoteController(); err != nil {
		return err
	}

	// step 2: deleteNVMeController if it exists
	if err := i.deleteNVMeController(); err != nil {
		return err
	}

	// step 3: deleteNVMeSubsystem
	if err := i.deleteNVMeSubsystem(); err != nil {
		return err
	}

	return waitForDeviceGone(i.devicePath)
}

// Create a controller with VirtioBlk transport information Bdev
func (i *opiInitiatorVirtioBlk) createVirtioBlk() (bdf string, err error) {
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	var blkDevice *opiapiStorage.VirtioBlk
	pf, vf, err := GetVirtioBlkAvailableFunction(i.kvmPciBridges)
	if err != nil {
		return "", fmt.Errorf("failed to detect free NVMe virtual function: %w", err)
	}

	physID := pf*32 + vf
	virtioBlkID := virtioBlkPrefix + strconv.Itoa(int(physID))
	bdf = fmt.Sprintf("0000:%02x:%02x.0", pf+1, vf)

	createReq := &opiapiStorage.CreateVirtioBlkRequest{
		VirtioBlk: &opiapiStorage.VirtioBlk{
			PcieId: &opiapiStorage.PciEndpoint{
				PhysicalFunction: int32(physID),
			},
			VolumeId: &opiapiCommon.ObjectKey{
				Value: i.volumeContext["model"],
			},
		},
		VirtioBlkId: virtioBlkID,
	}
	klog.Infof("OPI.CreateVirtioBlkRequest(...) => %s", createReq)
	blkDevice, err = i.frontendVirtioBlkClient.CreateVirtioBlk(ctxTimeout, createReq)
	if err != nil {
		return "", fmt.Errorf("failed to create virtio-blk device with pfId (%d) error: %w", physID, err)
	}
	i.virtioBlkName = blkDevice.Name

	return bdf, nil
}

// Delete the controller with VirtioBlk transport information Bdev
func (i *opiInitiatorVirtioBlk) deleteVirtioBlk() error {
	if i.virtioBlkName == "" {
		return nil
	}
	ctxTimeout, cancel := i.ctxTimeout()
	defer cancel()

	// DeleteVirtioBlk, with "AllowMissing: true", deleting operation will always succeed even the resource is not found
	deleteReq := &opiapiStorage.DeleteVirtioBlkRequest{
		Name:         i.virtioBlkName,
		AllowMissing: true,
	}
	klog.Infof("OPI.DeleteVirtioBlkRequest(...) => %s", deleteReq)

	_, err := i.frontendVirtioBlkClient.DeleteVirtioBlk(ctxTimeout, deleteReq)
	if err != nil {
		klog.Errorf("OPI.Nvme DeleteVirtioBlk error: %s", err)
		return err
	}
	klog.Info("OPI.DeleteVirtioBlk successfully")
	i.virtioBlkName = ""

	return nil
}

// For OPI VirtioBlk Connect(), two steps will be included.
// The first step is to connect to the remote controller, this step is used to connect to the storage node.
// The second step is CreateVirtioBlk, which is calling vhost_create_blk_controller on xPU node.
// After these two steps, a VirtioBlk device will appear.
func (i *opiInitiatorVirtioBlk) Connect() (string, error) {
	// step 1: connect to remote controller
	err := i.createNvmfRemoteController()
	if err != nil {
		return "", err
	}

	// step 2: Create a controller with virtio_blk transport information Bdev
	bdf, err := i.createVirtioBlk()
	if err != nil {
		if delErr := i.deleteNvmfRemoteController(); delErr != nil {
			klog.Errorf("Failed to clean nvme remote controller: %v", delErr)
		}
		return "", err
	}

	// find the dir file path for the virtio device
	devicePath, err := GetVirtioBlkDevice(bdf)
	if err != nil {
		if delErr := i.deleteVirtioBlk(); delErr != nil {
			klog.Errorf("Failed to clean block device: %v", delErr)
		}
		if delErr := i.deleteNvmfRemoteController(); delErr != nil {
			klog.Errorf("Failed to clean nvme remote controller: %v", delErr)
		}
		return "", err
	}

	i.devicePath = devicePath

	return devicePath, nil
}

// For OPI VirtioBlk Disconnect(), two steps will be included, namely DeleteVirtioBlk and DeleteNVMfRemoteController.
func (i *opiInitiatorVirtioBlk) Disconnect() error {
	// DeleteVirtioBlk if it exists
	if i.virtioBlkName == "" {
		klog.Info("No virtio block device ")
	}
	if err := i.deleteVirtioBlk(); err != nil {
		return err
	}

	// DeleteNVMfRemoteController
	if err := i.deleteNvmfRemoteController(); err != nil {
		return err
	}

	if err := waitForDeviceGone(i.devicePath); err != nil {
		klog.Errorf("Expected the block device '%s' gets deleted", i.devicePath)
	}
	i.devicePath = ""
	return nil
}
