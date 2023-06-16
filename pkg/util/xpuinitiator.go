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
	"strings"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog"
)

const (
	xpuNvmfTCPTargetType = "tcp"
	xpuNvmfTCPAdrFam     = "ipv4"
	xpuNvmfTCPTargetAddr = "127.0.0.1"
	xpuNvmfTCPTargetPort = "4421"
	xpuNvmfTCPSubNqnPref = "nqn.2022-04.io.spdk.csi:cnode0:uuid:"
)

type TransportType string

const (
	TransportTypeNvmfTCP   = "nvmftcp"
	TransportTypeNvme      = "nvme"
	TransportTypeVirtioBlk = "virtioblk"
)

type XpuInitiator interface {
	Connect(context.Context, *ConnectParams) error
	Disconnect(context.Context) error
}

type ConnectParams struct {
	tcpTargetPort string // NvmfTCP
	vPf           uint32 // VirtiioBlk, and Vfiouser
}

type XpuTargetType struct {
	Backend string
	TrType  TransportType
}

type xpuInitiator struct {
	backend       XpuInitiator
	targetInfo    *XpuTargetType
	volumeContext map[string]string
	kvmPciBridges int
	timeout       time.Duration
	bdf           string // in case of VirtioBlk and Vfiouser(Nvme)
}

var _ SpdkCsiInitiator = &xpuInitiator{}

func NewSpdkCsiXpuInitiator(volumeContext map[string]string, xpuConnClient *grpc.ClientConn, xpuTargetType string, kvmPciBridges int) (SpdkCsiInitiator, error) {
	targetInfo, err := parseSpdkXpuTargetType(xpuTargetType)
	if err != nil {
		return nil, err
	}
	xpu := &xpuInitiator{
		targetInfo:    targetInfo,
		volumeContext: volumeContext,
		kvmPciBridges: kvmPciBridges,
		timeout:       60 * time.Second,
	}
	var backend XpuInitiator
	switch targetInfo.Backend {
	case "sma":
		backend, err = NewSpdkCsiSmaInitiator(volumeContext, xpuConnClient, targetInfo.TrType)
	default:
		return nil, fmt.Errorf("unknown target type: %q", targetInfo.Backend)
	}
	if err != nil {
		return nil, err
	}
	xpu.backend = backend
	return xpu, nil
}

func (xpu *xpuInitiator) Connect( /*ctx *context.Context*/ ) (string, error) {
	ctx, cancel := xpu.ctxTimeout()
	defer cancel()

	switch xpu.targetInfo.TrType {
	case TransportTypeNvmfTCP:
		return xpu.ConnectNvmfTCP(ctx)
	case TransportTypeNvme:
		return xpu.ConnectNvme(ctx)
	case TransportTypeVirtioBlk:
		return xpu.ConnectVirtioBlk(ctx)
	}

	return "", fmt.Errorf("unsupported xpu transport type %q", xpu.targetInfo.TrType)
}

func (xpu *xpuInitiator) ConnectNvmfTCP(ctx context.Context) (string, error) {
	if err := xpu.backend.Connect(ctx, &ConnectParams{tcpTargetPort: smaNvmfTCPTargetPort}); err != nil {
		return "", err
	}

	// Initiate target connection with cmd:
	//   nvme connect -t tcp -a "127.0.0.1" -s 4421 -n "nqn.2022-04.io.spdk.csi:cnode0:uuid:*"
	//nolint: contextcheck, gocritic
	devicePath, err := newInitiatorNVMf(xpu.volumeContext["model"]).Connect()
	if err != nil {
		// Call Disconnect(), to clean up if nvme connect failed, while CreateDevice and AttachVolume succeeded
		if errx := xpu.backend.Disconnect(ctx); errx != nil {
			klog.Errorf("clean up error: %s", errx)
		}
		return "", err
	}

	return devicePath, nil
}

// ConnectNvme connects to nvme volume using one of 'vfiouser' transport protocol.
// It uses for the available PCI bridge function for connecting the device.
// On success it returns the block device path on host.
func (xpu *xpuInitiator) ConnectNvme(ctx context.Context) (string, error) {
	devicePath, err := CheckIfNvmeDeviceExists(xpu.volumeContext["model"], nil)
	if devicePath != "" {
		klog.Infof("Found existing device for '%s': %v", xpu.volumeContext["mode"], devicePath)
		return devicePath, nil
	}
	if !os.IsNotExist(err) {
		klog.Errorf("failed to detect existing nvme device for '%s'", xpu.volumeContext["model"])
	}
	pf, vf, err := GetNvmeAvailableFunction(xpu.kvmPciBridges)
	if err != nil {
		return "", fmt.Errorf("failed to detect free NVMe virtual function: %w", err)
	}
	// SMA always expects Vf value as 0. It detects the right KVM bus from Pf value.
	vPf := pf*32 + vf
	klog.Infof("Using next available function: %d", vf)

	// Ask the backed to connect to the volume
	if err = xpu.backend.Connect(ctx, &ConnectParams{vPf: vPf}); err != nil {
		return "", err
	}

	bdf := fmt.Sprintf("0000:%02x:%02x.0", pf+1, vf)
	klog.Infof("Waiting still device ready for '%s' at '%s' ...", xpu.volumeContext["model"], bdf)
	devicePath, err = GetNvmeDeviceName(xpu.volumeContext["model"], bdf)
	if err != nil {
		klog.Errorf("Could not detect the device: %s", err)
		if errx := xpu.backend.Disconnect(ctx); errx != nil {
			klog.Errorf("failed to disconnect device: %v", err)
		}
		return "", err
	}
	xpu.bdf = bdf

	return devicePath, nil
}

// ConnectVirtioBlk connects to nvme volume using one of 'virtio-blk' transport protocol.
// It uses for the available PCI bridge function for connecting the device.
// On success it returns the block device path on host.
func (xpu *xpuInitiator) ConnectVirtioBlk(ctx context.Context) (string, error) {
	pf, vf, err := GetVirtioBlkAvailableFunction(xpu.kvmPciBridges)
	if err != nil {
		return "", fmt.Errorf("failed to detect free NVMe virtual function: %w", err)
	}
	// SMA always expects Vf value as 0. It detects the right KVM bus from Pf value.
	vPf := pf*32 + vf
	klog.Infof("Using next available function: %d", vf)

	// Ask the backed to connect to the volume
	if err = xpu.backend.Connect(ctx, &ConnectParams{vPf: vPf}); err != nil {
		return "", err
	}

	bdf := fmt.Sprintf("0000:%02x:%02x.0", pf+1, vf)
	klog.Infof("Waiting still device ready for '%s' at '%s' ...", xpu.volumeContext["model"], bdf)
	var devicePath string
	devicePath, err = GetVirtioBlkDevice(bdf, true)
	if err != nil {
		klog.Errorf("Could not detect the device: %s", err)
		if errx := xpu.backend.Disconnect(ctx); errx != nil {
			klog.Errorf("failed to disconnect device: %v", errx)
		}
		return "", err
	}
	xpu.bdf = bdf

	return devicePath, nil
}

func (xpu *xpuInitiator) Disconnect( /*ctx context.Context*/ ) error {
	ctx, cancel := xpu.ctxTimeout()
	defer cancel()

	switch xpu.targetInfo.TrType {
	case TransportTypeNvmfTCP:
		return xpu.DisconnectNvmfTCP(ctx)
	case TransportTypeNvme:
		return xpu.DisconnectNvme(ctx)
	case TransportTypeVirtioBlk:
		return xpu.DisconnectVirtioBlk(ctx)
	}

	return fmt.Errorf("unsupported xpu transport type %q", xpu.targetInfo.TrType)
}

// DisconnectNvmTCP disconnects volume. First it executes "nvme disconnect"
// to terminate the target connection and then ask the backedn to drop the
// device.
func (xpu *xpuInitiator) DisconnectNvmfTCP(ctx context.Context) error {
	// nvme disconnect -n "nqn.2022-04.io.spdk.csi:cnode0:uuid:*"
	//nolint: contextcheck, gocritic
	if err := newInitiatorNVMf(xpu.volumeContext["model"]).Disconnect(); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	return xpu.backend.Disconnect(ctx)
}

// DisconnectNvme disconnects the target nvme device
func (xpu *xpuInitiator) DisconnectNvme(ctx context.Context) error {
	devicePath, err := CheckIfNvmeDeviceExists(xpu.volumeContext["model"], nil)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := xpu.backend.Disconnect(ctx); err != nil {
		return err
	}

	return waitForDeviceGone(devicePath)
}

// DisconnectVirtioBlk disconnects the target virtio-blk device
func (xpu *xpuInitiator) DisconnectVirtioBlk(ctx context.Context) error {
	devicePath, err := GetVirtioBlkDevice(xpu.bdf, false)
	if err != nil {
		return fmt.Errorf("failed to get block device path at bdf '%s': %w", xpu.bdf, err)
	}
	if err := xpu.backend.Disconnect(ctx); err != nil {
		return err
	}

	return waitForDeviceGone(devicePath)
}

func parseSpdkXpuTargetType(xpuTargetType string) (*XpuTargetType, error) {
	parts := strings.Split(xpuTargetType, "-")
	if parts[0] != "xpu" || len(parts) != 3 {
		return nil, fmt.Errorf("invalid xpuTargetType %q", xpuTargetType)
	}

	return &XpuTargetType{Backend: parts[1], TrType: TransportType(parts[2])}, nil
}

func (xpu *xpuInitiator) ctxTimeout() (context.Context, context.CancelFunc) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), xpu.timeout)
	return ctxTimeout, cancel
}

// re-use the Connect() and Disconnect() functions from initiator.go
func newInitiatorNVMf(model string) *initiatorNVMf {
	return &initiatorNVMf{
		targetType: xpuNvmfTCPTargetType,
		targetAddr: xpuNvmfTCPTargetAddr,
		targetPort: xpuNvmfTCPTargetPort,
		nqn:        xpuNvmfTCPSubNqnPref + model,
		model:      model,
	}
}
