/*
Copyright 2017 The Kubernetes Authors.

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

package azurefile

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/kubernetes/pkg/cloudprovider/providers/azure"
	"k8s.io/kubernetes/pkg/volume/util"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/andyzhangx/azurefile-csi-driver/pkg/csi-common"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"github.com/pborman/uuid"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	volumeIDTemplate = "%s#%s#%s"
)

type controllerServer struct {
	*csicommon.DefaultControllerServer
	cloud *azure.Cloud
}

func (cs *controllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.Errorf("invalid create volume req: %v", req)
		return nil, err
	}

	// Validate arguments
	volumeCapabilities := req.GetVolumeCapabilities()
	name := req.GetName()
	if len(name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Name must be provided")
	}
	if volumeCapabilities == nil || len(volumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Volume capabilities must be provided")
	}

	volSizeBytes := int64(req.GetCapacityRange().GetRequiredBytes())
	requestGiB := int(util.RoundUpSize(volSizeBytes, 1024*1024*1024))

	parameters := req.GetParameters()
	var sku, resourceGroup, location, account string

	// Apply ProvisionerParameters (case-insensitive). We leave validation of
	// the values to the cloud provider.
	for k, v := range parameters {
		switch strings.ToLower(k) {
		case "skuname":
			sku = v
		case "location":
			location = v
		case "storageaccount":
			account = v
		case "resourcegroup":
			resourceGroup = v
		default:
			return nil, fmt.Errorf("invalid option %q", k)
		}
	}

	// when use azure file premium, account kind should be specified as FileStorage
	accountKind := string(storage.StorageV2)
	if strings.HasPrefix(strings.ToLower(sku), "premium") {
		accountKind = string(storage.FileStorage)
	}

	// dynamic provisioning since storage account is not provided by secrets
	// File share name has a length limit of 63, and it cannot contain two consecutive '-'s.
	// todo: get cluster name
	fileShareName := util.GenerateVolumeName("pvc-file", uuid.NewUUID().String(), 63)
	fileShareName = strings.Replace(fileShareName, "--", "-", -1)

	glog.V(2).Infof("begin to create file share(%s) on account(%s) type(%s) rg(%s) location(%s) size(%d)", fileShareName, account, sku, resourceGroup, location, requestGiB)
	retAccount, _, err := cs.cloud.CreateFileShare(fileShareName, account, sku, accountKind, resourceGroup, location, requestGiB)
	if err != nil {
		return nil, fmt.Errorf("failed to create file share(%s) on account(%s) type(%s) rg(%s) location(%s) size(%d)", fileShareName, account, sku, resourceGroup, location, requestGiB)
	}
	volumeID := fmt.Sprintf(volumeIDTemplate, resourceGroup, retAccount, fileShareName)

	if req.GetVolumeContentSource() != nil {
		contentSource := req.GetVolumeContentSource()
		if contentSource.GetSnapshot() != nil {
		}
	}
	glog.V(2).Infof("create file share %s on storage account %s successfully", fileShareName, retAccount)

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id:            volumeID,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			Attributes:    parameters,
		},
	}, nil
}

func (cs *controllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		return nil, fmt.Errorf("invalid delete volume req: %v", req)
	}
	volumeID := req.VolumeId

	resourceGroupName, accountName, fileShareName, err := getFileShareInfo(volumeID)
	if err != nil {
		return nil, err
	}

	if resourceGroupName == "" {
		resourceGroupName = cs.cloud.ResourceGroup
	}

	accountKey, err := GetStorageAccesskey(cs.cloud, accountName, resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("no key for storage account(%s) under resource group(%s), err %v", accountName, resourceGroupName, err)
	}

	glog.V(2).Infof("deleting azure file(%s) rg(%s) account(%s) volumeID(%s)", fileShareName, resourceGroupName, accountName, volumeID)
	if err := cs.cloud.DeleteFileShare(accountName, accountKey, fileShareName); err != nil {
		return nil, err
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *controllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}
	if _, ok := azureFileVolumes[req.GetVolumeId()]; !ok {
		return nil, status.Error(codes.NotFound, "Volume does not exist")
	}

	for _, cap := range req.VolumeCapabilities {
		if cap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{Supported: false, Message: ""}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{Supported: true, Message: ""}, nil
}
