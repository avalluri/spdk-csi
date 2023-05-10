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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	opiapiStorage "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
)

// MockNVMfRemoteControllerServiceClient is a mock implementation of the NVMfRemoteControllerServiceClient interface
type MockNVMfRemoteControllerServiceClient struct {
	mock.Mock
}

func (m *MockNVMfRemoteControllerServiceClient) CreateNVMfRemoteController(ctx context.Context, in *opiapiStorage.CreateNVMfRemoteControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NVMfRemoteController, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.NVMfRemoteController), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (m *MockNVMfRemoteControllerServiceClient) DeleteNVMfRemoteController(ctx context.Context, in *opiapiStorage.DeleteNVMfRemoteControllerRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*emptypb.Empty), args.Error(1) //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (m *MockNVMfRemoteControllerServiceClient) UpdateNVMfRemoteController(ctx context.Context, in *opiapiStorage.UpdateNVMfRemoteControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NVMfRemoteController, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.NVMfRemoteController), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (m *MockNVMfRemoteControllerServiceClient) ListNVMfRemoteControllers(ctx context.Context, in *opiapiStorage.ListNVMfRemoteControllersRequest, _ ...grpc.CallOption) (*opiapiStorage.ListNVMfRemoteControllersResponse, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.ListNVMfRemoteControllersResponse), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (m *MockNVMfRemoteControllerServiceClient) GetNVMfRemoteController(ctx context.Context, in *opiapiStorage.GetNVMfRemoteControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NVMfRemoteController, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return nil, args.Error(1)
}

func (m *MockNVMfRemoteControllerServiceClient) NVMfRemoteControllerReset(ctx context.Context, in *opiapiStorage.NVMfRemoteControllerResetRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return args.Get(0).(*emptypb.Empty), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (m *MockNVMfRemoteControllerServiceClient) NVMfRemoteControllerStats(ctx context.Context, in *opiapiStorage.NVMfRemoteControllerStatsRequest, _ ...grpc.CallOption) (*opiapiStorage.NVMfRemoteControllerStatsResponse, error) { //nolint: lll, gocritic
	args := m.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.NVMfRemoteControllerStatsResponse), args.Error(1) //nolint: forcetypeassert, gocritic
}

type MockFrontendVirtioBlkServiceClient struct {
	mock.Mock
}

func (i *MockFrontendVirtioBlkServiceClient) CreateVirtioBlk(ctx context.Context, in *opiapiStorage.CreateVirtioBlkRequest, _ ...grpc.CallOption) (*opiapiStorage.VirtioBlk, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.VirtioBlk), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendVirtioBlkServiceClient) DeleteVirtioBlk(ctx context.Context, in *opiapiStorage.DeleteVirtioBlkRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*emptypb.Empty), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendVirtioBlkServiceClient) UpdateVirtioBlk(ctx context.Context, in *opiapiStorage.UpdateVirtioBlkRequest, _ ...grpc.CallOption) (*opiapiStorage.VirtioBlk, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.VirtioBlk), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (i *MockFrontendVirtioBlkServiceClient) ListVirtioBlks(ctx context.Context, in *opiapiStorage.ListVirtioBlksRequest, _ ...grpc.CallOption) (*opiapiStorage.ListVirtioBlksResponse, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.ListVirtioBlksResponse), args.Error(1) //nolint: forcetypeassert, gocritic
}

func (i *MockFrontendVirtioBlkServiceClient) GetVirtioBlk(ctx context.Context, in *opiapiStorage.GetVirtioBlkRequest, _ ...grpc.CallOption) (*opiapiStorage.VirtioBlk, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.VirtioBlk), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendVirtioBlkServiceClient) VirtioBlkStats(ctx context.Context, in *opiapiStorage.VirtioBlkStatsRequest, _ ...grpc.CallOption) (*opiapiStorage.VirtioBlkStatsResponse, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	return args.Get(0).(*opiapiStorage.VirtioBlkStatsResponse), args.Error(1) //nolint: forcetypeassert, gocritic
}

type MockFrontendNvmeServiceClient struct {
	mock.Mock
}

func (i *MockFrontendNvmeServiceClient) CreateNvmeSubsystem(ctx context.Context, in *opiapiStorage.CreateNvmeSubsystemRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeSubsystem, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeSubsystem), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) DeleteNvmeSubsystem(ctx context.Context, in *opiapiStorage.DeleteNvmeSubsystemRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*emptypb.Empty), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) UpdateNvmeSubsystem(_ context.Context, _ *opiapiStorage.UpdateNvmeSubsystemRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeSubsystem, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) ListNvmeSubsystems(_ context.Context, _ *opiapiStorage.ListNvmeSubsystemsRequest, _ ...grpc.CallOption) (*opiapiStorage.ListNvmeSubsystemsResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) GetNvmeSubsystem(ctx context.Context, in *opiapiStorage.GetNvmeSubsystemRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeSubsystem, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeSubsystem), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) NvmeSubsystemStats(_ context.Context, _ *opiapiStorage.NvmeSubsystemStatsRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeSubsystemStatsResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) CreateNvmeController(ctx context.Context, in *opiapiStorage.CreateNvmeControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeController, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeController), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) DeleteNvmeController(ctx context.Context, in *opiapiStorage.DeleteNvmeControllerRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*emptypb.Empty), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) UpdateNvmeController(_ context.Context, _ *opiapiStorage.UpdateNvmeControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeController, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) ListNvmeControllers(_ context.Context, _ *opiapiStorage.ListNvmeControllersRequest, _ ...grpc.CallOption) (*opiapiStorage.ListNvmeControllersResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) GetNvmeController(ctx context.Context, in *opiapiStorage.GetNvmeControllerRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeController, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeController), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) NvmeControllerStats(_ context.Context, _ *opiapiStorage.NvmeControllerStatsRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeControllerStatsResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) CreateNvmeNamespace(ctx context.Context, in *opiapiStorage.CreateNvmeNamespaceRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeNamespace, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeNamespace), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) DeleteNvmeNamespace(ctx context.Context, in *opiapiStorage.DeleteNvmeNamespaceRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*emptypb.Empty), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) UpdateNvmeNamespace(_ context.Context, _ *opiapiStorage.UpdateNvmeNamespaceRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeNamespace, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) ListNvmeNamespaces(_ context.Context, _ *opiapiStorage.ListNvmeNamespacesRequest, _ ...grpc.CallOption) (*opiapiStorage.ListNvmeNamespacesResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func (i *MockFrontendNvmeServiceClient) GetNvmeNamespace(ctx context.Context, in *opiapiStorage.GetNvmeNamespaceRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeNamespace, error) { //nolint: lll, gocritic
	args := i.Called(ctx, in)
	if args.Get(0) != nil {
		return args.Get(0).(*opiapiStorage.NvmeNamespace), nil //nolint: forcetypeassert, gocritic
	}
	return nil, args.Error(1)
}

func (i *MockFrontendNvmeServiceClient) NvmeNamespaceStats(_ context.Context, _ *opiapiStorage.NvmeNamespaceStatsRequest, _ ...grpc.CallOption) (*opiapiStorage.NvmeNamespaceStatsResponse, error) { //nolint: lll, gocritic
	return nil, nil
}

func TestCreateNvmfRemoteController(t *testing.T) {
	// Create a mock NVMfRemoteControllerServiceClient
	mockClient := new(MockNVMfRemoteControllerServiceClient)

	// Create an instance of opiCommon
	volumeContext := map[string]string{
		"targetPort": "1234",
		"targetAddr": "192.168.0.1",
		"nqn":        "nqn-value",
		"model":      "model-value",
	}
	opi := &opiCommon{
		volumeContext:              volumeContext,
		timeout:                    time.Second,
		kvmPciBridges:              0,
		opiClient:                  nil,
		nvmfRemoteControllerClient: mockClient,
		devicePath:                 "",
	}

	controllerID := nvmfRemoteControllerPrefix + volumeContext["model"]
	// Mock the CreateNVMfRemoteController function to return a response with the specified ID
	mockClient.On("CreateNVMfRemoteController", mock.Anything, mock.Anything).Return(&opiapiStorage.NVMfRemoteController{
		Name: controllerID,
	}, nil)

	// Mock the GetNVMfRemoteController function to return a nil response and an error
	mockClient.On("GetNVMfRemoteController", mock.Anything, mock.Anything).Return(nil, errors.New("Controller does not exist"))

	// Set the necessary volume context values

	// Call the function under test
	err := opi.createNvmfRemoteController()
	// Assert that the error returned is nil
	assert.NoError(t, err, "create remote nvmf controller")

	// Assert that the GetNVMfRemoteController and CreateNVMfRemoteController functions were called with the expected arguments
	mockClient.AssertCalled(t, "GetNVMfRemoteController", mock.Anything, &opiapiStorage.GetNVMfRemoteControllerRequest{Name: controllerID})
	mockClient.AssertCalled(t, "CreateNVMfRemoteController", mock.Anything, &opiapiStorage.CreateNVMfRemoteControllerRequest{
		NvMfRemoteController: &opiapiStorage.NVMfRemoteController{
			Name:    controllerID,
			Trtype:  opiapiStorage.NvmeTransportType_NVME_TRANSPORT_TCP,
			Adrfam:  opiapiStorage.NvmeAddressFamily_NVMF_ADRFAM_IPV4,
			Traddr:  "192.168.0.1",
			Trsvcid: 1234,
			Subnqn:  "nqn-value",
			Hostnqn: "nqn.2023-04.io.spdk.csi:remote.controller:uuid:" + opi.volumeContext["model"],
		},
	})
}

func TestDeleteNvmfRemoteController_Success(t *testing.T) {
	// Create a mock client
	mockClient := new(MockNVMfRemoteControllerServiceClient)

	// Create the OPI instance with the mock client
	opi := &opiCommon{
		nvmfRemoteControllerClient: mockClient,
		volumeContext: map[string]string{
			"model": "model-value",
		},
	}

	// Set up expectations
	nvmfRemoteControllerID := nvmfRemoteControllerPrefix + opi.volumeContext["model"]
	deleteReq := &opiapiStorage.DeleteNVMfRemoteControllerRequest{
		Name:         nvmfRemoteControllerID,
		AllowMissing: true,
	}
	mockClient.On("DeleteNVMfRemoteController", mock.Anything, deleteReq).Return(&emptypb.Empty{}, nil)

	// Call the method under test
	err := opi.deleteNvmfRemoteController()

	// Assert that the expected methods were called
	mockClient.AssertExpectations(t)

	// Assert that no error was returned
	assert.NoError(t, err)
}

func TestDeleteNvmfRemoteController_Error(t *testing.T) {
	// Create a mock client
	mockClient := new(MockNVMfRemoteControllerServiceClient)

	// Create the OPI instance with the mock client
	opi := &opiCommon{
		nvmfRemoteControllerClient: mockClient,
		volumeContext: map[string]string{
			"model": "model-value",
		},
	}

	// Set up expectations
	nvmfRemoteControllerID := nvmfRemoteControllerPrefix + opi.volumeContext["model"]
	deleteReq := &opiapiStorage.DeleteNVMfRemoteControllerRequest{
		Name:         nvmfRemoteControllerID,
		AllowMissing: true,
	}
	expectedErr := errors.New("delete error")
	mockClient.On("DeleteNVMfRemoteController", mock.Anything, deleteReq).Return(nil, expectedErr)

	// Call the method under test
	err := opi.deleteNvmfRemoteController()

	// Assert that the expected methods were called
	mockClient.AssertExpectations(t)

	// Assert that the correct error was returned
	assert.ErrorContains(t, err, expectedErr.Error())
}

func TestCreateVirtioBlk_Failure(t *testing.T) {
	// Create a mock client
	mockClient := new(MockFrontendVirtioBlkServiceClient)

	// Create a mock NVMfRemoteControllerServiceClient
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	// Create the OPI instance with the mock client
	opi := &opiInitiatorVirtioBlk{
		frontendVirtioBlkClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}

	// Mock the CreateNVMfRemoteController function to return a response with the specified ID
	mockClient.On("GetVirtioBlk", mock.Anything, mock.Anything).Return(nil, errors.New("Could not find Controller"))

	// Mock the GetNVMfRemoteController function to return a nil response and an error
	mockClient.On("CreateVirtioBlk", mock.Anything, mock.Anything).Return(nil, errors.New("Controller does not exist"))

	bdf, err := opi.createVirtioBlk()
	assert.Equal(t, bdf, "")
	assert.NotEqual(t, err, nil)
}

func TestCreateVirtioBlk_Success(t *testing.T) {
	// Create a mock client
	mockClient := new(MockFrontendVirtioBlkServiceClient)

	// Create a mock NVMfRemoteControllerServiceClient
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	// Create the OPI instance with the mock client
	opi := &opiInitiatorVirtioBlk{
		frontendVirtioBlkClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{"model": "xxx"},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}

	// Mock the CreateNVMfRemoteController function to return a response with the specified ID
	mockClient.On("GetVirtioBlk", mock.Anything, mock.Anything).Return(nil, errors.New("Could not find Controller"))

	// Mock the GetNVMfRemoteController function to return a nil response and an error
	mockClient.On("CreateVirtioBlk", mock.Anything, mock.Anything).Return(
		&opiapiStorage.VirtioBlk{
			Name: virtioBlkPrefix + opi.volumeContext["mode"],
		}, nil)

	bdf, err := opi.createVirtioBlk()
	assert.Equal(t, bdf, "0000:01:00.0")
	assert.Equal(t, err, nil)
}

func TestDeleteVirtioBlk_Success(t *testing.T) {
	// Create a mock client
	mockClient := new(MockFrontendVirtioBlkServiceClient)

	// Create a mock NVMfRemoteControllerServiceClient
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	// Create the OPI instance with the mock client
	opi := &opiInitiatorVirtioBlk{
		frontendVirtioBlkClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{"model": "xxx"},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}

	// Mock the CreateNVMfRemoteController function to return a response with the specified ID
	mockClient.On("DeleteVirtioBlk", mock.Anything, mock.Anything).Return(nil, errors.New("Could not find Controller"))

	err := opi.deleteVirtioBlk()
	assert.Equal(t, err, nil)
}

func TestDeleteVirtioBlk_Failure(t *testing.T) {
	// Create a mock client
	mockClient := new(MockFrontendVirtioBlkServiceClient)

	// Create a mock NVMfRemoteControllerServiceClient
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	// Create the OPI instance with the mock client
	opi := &opiInitiatorVirtioBlk{
		frontendVirtioBlkClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{"model": "xxx"},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}

	// Mock the CreateNVMfRemoteController function to return a response with the specified ID
	mockClient.On("DeleteVirtioBlk", mock.Anything, mock.Anything).Return(nil, errors.New("failed to delete device"))

	err := opi.deleteVirtioBlk()
	assert.Equal(t, err, nil)
}

func TestCreateNvmeSubsystem(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{"model": "xxx"},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("GetNvmeSubsystem", mock.Anything, mock.Anything).Return(nil,
		status.Error(codes.NotFound, "subsystem not found"))
	mockClient.On("CreateNvmeSubsystem", mock.Anything, mock.Anything).Return(
		&opiapiStorage.NvmeSubsystem{
			Name: nvmeSubSystemPrefix + opi.volumeContext["model"],
		}, nil)

	err := opi.createNVMeSubsystem()
	assert.Equal(t, err, nil)
}

func TestDeleteNvmeSubsystem(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("DeleteNvmeSubsystem", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	err := opi.deleteNVMeSubsystem()
	assert.NotEqual(t, err, nil)
}

func TestCreateNvmeController(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{"model": "xxx"},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("GetNvmeController", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	mockClient.On("CreateNvmeController", mock.Anything, mock.Anything).Return(
		&opiapiStorage.NvmeController{
			Name: nvmeControllerPrefix + opi.volumeContext["model"],
		}, nil)
	_, err := opi.createNVMeController()
	assert.Equal(t, err, nil)
}

func TestDeleteNvmeController(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("DeleteNvmeController", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	err := opi.deleteNVMeController()
	assert.NotEqual(t, err, nil)
}

func TestCreateNvmeNamespace(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("GetNvmeNamespace", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	mockClient.On("CreateNvmeNamespace", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	err := opi.createNVMeNamespace()
	assert.NotEqual(t, err, nil)
}

func TestDeleteNvmeNamespace(t *testing.T) {
	mockClient := new(MockFrontendNvmeServiceClient)
	mockClient2 := new(MockNVMfRemoteControllerServiceClient)
	opi := &opiInitiatorNvme{
		frontendNvmeClient: mockClient,
		// Create an instance of opiCommon
		opiCommon: &opiCommon{
			volumeContext:              map[string]string{},
			timeout:                    time.Second,
			kvmPciBridges:              1,
			opiClient:                  nil,
			nvmfRemoteControllerClient: mockClient2,
			devicePath:                 "",
		},
	}
	mockClient.On("DeleteNvmeNamespace", mock.Anything, mock.Anything).Return(nil, errors.New("unable to find key"))
	err := opi.deleteNVMeNamespace()
	assert.NotEqual(t, err, nil)
}
