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
	"net/url"
	"strings"
	"time"

	volumehelper "sigs.k8s.io/azurefile-csi-driver/pkg/util"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes"
	"github.com/pborman/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

// CreateVolume provisions an azure file
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(2).Infof("CreateVolume called with request %v", *req)
	if err := d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		klog.Errorf("invalid create volume req: %v", req)
		return nil, err
	}

	volumeCapabilities := req.GetVolumeCapabilities()
	name := req.GetName()
	if len(name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Name must be provided")
	}
	if len(volumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Volume capabilities must be provided")
	}

	capacityBytes := req.GetCapacityRange().GetRequiredBytes()
	requestGiB := volumehelper.RoundUpGiB(capacityBytes)
	if requestGiB == 0 {
		requestGiB = defaultAzureFileQuota
		klog.Warningf("no quota specified, set as default value(%d GiB)", defaultAzureFileQuota)
	}

	parameters := req.GetParameters()
	var sku, resourceGroup, location, account, fileShareName, diskName, fsType string

	// Apply ProvisionerParameters (case-insensitive). We leave validation of
	// the values to the cloud provider.
	for k, v := range parameters {
		switch strings.ToLower(k) {
		case "skuname":
			sku = v
		case "storageaccounttype":
			sku = v
		case "location":
			location = v
		case "storageaccount":
			account = v
		case "resourcegroup":
			resourceGroup = v
		case shareNameField:
			fileShareName = v
		case diskNameField:
			diskName = v
		case fsTypeField:
			fsType = v
		default:
			return nil, fmt.Errorf("invalid option %q", k)
		}
	}

	fileShareSize := int(requestGiB)
	// when use azure file premium, account kind should be specified as FileStorage
	accountKind := string(storage.StorageV2)
	if strings.HasPrefix(strings.ToLower(sku), "premium") {
		accountKind = string(storage.FileStorage)
		if fileShareSize < minimumPremiumShareSize {
			fileShareSize = minimumPremiumShareSize
		}
	}

	if fileShareName == "" {
		fileShareName = getValidFileShareName(name)
	}

	klog.V(2).Infof("begin to create file share(%s) on account(%s) type(%s) rg(%s) location(%s) size(%d)", fileShareName, account, sku, resourceGroup, location, fileShareSize)

	var retAccount, retAccountKey string
	err := wait.Poll(1*time.Second, 3*time.Minute, func() (bool, error) {
		var retErr error
		retAccount, retAccountKey, retErr = d.cloud.CreateFileShare(fileShareName, account, sku, accountKind, resourceGroup, location, fileShareSize)
		if retErr != nil {
			if strings.Contains(retErr.Error(), accountNotProvisioned) {
				klog.Warningf("CreateFileShare failed with %s error, sleep 1s to retry", accountNotProvisioned)
				time.Sleep(time.Second)
				return false, nil
			}
		}
		return true, retErr
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create file share(%s) on account(%s) type(%s) rg(%s) location(%s) size(%d), error: %v", fileShareName, account, sku, resourceGroup, location, fileShareSize, err)
	}
	if retAccount == "" || retAccountKey == "" {
		return nil, fmt.Errorf("create file share(%s) on account(%s) type(%s) rg(%s) location(%s) size(%d) timeout(3m)", fileShareName, account, sku, resourceGroup, location, fileShareSize)
	}
	klog.V(2).Infof("create file share %s on storage account %s successfully", fileShareName, retAccount)

	if err := d.checkFileShareCapacity(retAccount, retAccountKey, fileShareName, fileShareSize); err != nil {
		return nil, err
	}

	isDiskMount := (fsType != "" && fsType != cifs)
	if isDiskMount && diskName == "" {
		diskName = uuid.NewUUID().String() + ".vhd"
		diskSizeBytes := volumehelper.GiBToBytes(requestGiB)
		klog.V(2).Infof("begin to create vhd file(%s) size(%d) on share(%s) on account(%s) type(%s) rg(%s) location(%s)",
			diskName, diskSizeBytes, fileShareName, account, sku, resourceGroup, location)
		if err := createDisk(ctx, retAccount, retAccountKey, d.cloud.Environment.StorageEndpointSuffix, fileShareName, diskName, diskSizeBytes); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create VHD disk: %v", err))
		}
		klog.V(2).Infof("create vhd file(%s) size(%d) on share(%s) on account(%s) type(%s) rg(%s) location(%s) successfully",
			diskName, diskSizeBytes, fileShareName, account, sku, resourceGroup, location)
		parameters[diskNameField] = diskName
	}

	volumeID := fmt.Sprintf(volumeIDTemplate, resourceGroup, retAccount, fileShareName, diskName)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			CapacityBytes: capacityBytes,
			VolumeContext: parameters,
		},
	}, nil
}

// DeleteVolume delete an azure file
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(2).Infof("DeleteVolume called with request %v", *req)
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if err := d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid delete volume request: %v", req)
	}

	volumeID := req.VolumeId
	shareURL, err := d.getShareURL(volumeID, req.GetSecrets())
	if err != nil {
		// According to CSI Driver Sanity Tester, should succeed when an invalid volume id is used
		klog.V(4).Infof("failed to get share url with (%s): %v, returning with success", volumeID, err)
		return &csi.DeleteVolumeResponse{}, nil
	}
	resourceGroupName, accountName, fileShareName, _, err := getFileShareInfo(volumeID)
	if err != nil {
		klog.Errorf("getFileShareInfo(%s) in DeleteVolume failed with error: %v", volumeID, err)
		return &csi.DeleteVolumeResponse{}, nil
	}

	if _, err = shareURL.Delete(ctx, azfile.DeleteSnapshotsOptionInclude); err != nil {
		return nil, status.Errorf(codes.Internal, "DeleteFileShare %s under %s failed with error: %v", fileShareName, accountName, err)
	}
	klog.V(2).Infof("azure file(%s) under rg(%s) account(%s) volume(%s) is deleted successfully", fileShareName, resourceGroupName, accountName, volumeID)

	return &csi.DeleteVolumeResponse{}, nil
}

// ValidateVolumeCapabilities return the capabilities of the volume
func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}

	volumeID := req.VolumeId
	resourceGroupName, accountName, _, fileShareName, _, err := d.getAccountInfo(volumeID, req.GetSecrets(), req.GetVolumeContext())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "error getting volume(%s) info: %v", volumeID, err)
	}
	if resourceGroupName == "" {
		resourceGroupName = d.cloud.ResourceGroup
	}
	if exists, err := d.checkFileShareExists(accountName, resourceGroupName, fileShareName); err != nil {
		return nil, status.Errorf(codes.NotFound, "error checking if volume(%s) exists: %v", volumeID, err)
	} else if !exists {
		return nil, status.Errorf(codes.NotFound, "the requested volume(%s) does not exist.", volumeID)
	}

	// azure file supports all AccessModes, no need to check capabilities here
	return &csi.ValidateVolumeCapabilitiesResponse{Message: ""}, nil
}

// ControllerGetCapabilities returns the capabilities of the Controller plugin
func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(2).Infof("ControllerGetCapabilities called with request %v", *req)

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: d.Cap,
	}, nil
}

// GetCapacity returns the capacity of the total available storage pool
func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes return all available volumes
func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerPublishVolume make a volume available on some required node
func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	klog.V(2).Infof("ControllerPublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	nodeID := req.GetNodeId()
	if len(nodeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Node ID not provided")
	}
	nodeName := types.NodeName(nodeID)
	if _, err := d.cloud.InstanceID(ctx, nodeName); err != nil {
		if err == cloudprovider.InstanceNotFound {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("failed to get azure instance id for node %q (%v)", nodeName, err))
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("get azure instance id for node %q failed with %v", nodeName, err))
	}

	_, accountName, accountKey, fileShareName, diskName, err := d.getAccountInfo(volumeID, req.GetSecrets(), req.GetVolumeContext())
	// always check diskName first since if it's not vhd disk attach, ControllerPublishVolume is not necessary
	if diskName == "" {
		klog.V(2).Infof("skip ControllerPublishVolume(%s) since it's not vhd disk attach", volumeID)
		return &csi.ControllerPublishVolumeResponse{}, nil
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("getAccountInfo(%s) failed with error: %v", volumeID, err))
	}

	accessMode := volCap.GetAccessMode()
	if accessMode == nil ||
		(accessMode.Mode != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER &&
			accessMode.Mode != csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported AccessMode(%v) for volume(%s)", volCap.GetAccessMode(), volumeID))
	}

	storageEndpointSuffix := d.cloud.Environment.StorageEndpointSuffix
	fileURL, err := getFileURL(accountName, accountKey, storageEndpointSuffix, fileShareName, diskName)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("getFileURL(%s,%s,%s,%s) returned with error: %v", accountName, storageEndpointSuffix, fileShareName, diskName, err))
	}
	if fileURL == nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("getFileURL(%s,%s,%s,%s) returned empty fileURL", accountName, storageEndpointSuffix, fileShareName, diskName))
	}

	d.volLockMap.LockEntry(volumeID)
	defer d.volLockMap.UnlockEntry(volumeID)
	properties, err := fileURL.GetProperties(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("GetProperties for volume(%s) on node(%s) returned with error: %v", volumeID, nodeID, err))
	}

	if v, ok := properties.NewMetadata()[metaDataNode]; ok {
		if v != "" {
			return nil, status.Error(codes.Internal, fmt.Sprintf("volume(%s) cannot be attached to node(%s) since it's already attached to node(%s)", volumeID, nodeID, v))
		}
	}

	if _, err = fileURL.SetMetadata(ctx, azfile.Metadata{metaDataNode: strings.ToLower(nodeID)}); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("SetMetadata for volume(%s) on node(%s) returned with error: %v", volumeID, nodeID, err))
	}
	klog.V(2).Infof("ControllerPublishVolume: volume(%s) attached to node(%s) successfully", volumeID, nodeID)
	return &csi.ControllerPublishVolumeResponse{}, nil
}

// ControllerUnpublishVolume detach the volume on a specified node
func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	klog.V(2).Infof("ControllerUnpublishVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID not provided")
	}

	nodeID := req.GetNodeId()
	if len(nodeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Node ID not provided")
	}

	_, accountName, accountKey, fileShareName, diskName, err := d.getAccountInfo(volumeID, req.GetSecrets(), map[string]string{})
	// always check diskName first since if it's not vhd disk detach, ControllerUnpublishVolume is not necessary
	if diskName == "" {
		klog.V(2).Infof("skip ControllerUnpublishVolume(%s) since it's not vhd disk detach", volumeID)
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("getAccountInfo(%s) failed with error: %v", volumeID, err))
	}

	storageEndpointSuffix := d.cloud.Environment.StorageEndpointSuffix
	fileURL, err := getFileURL(accountName, accountKey, storageEndpointSuffix, fileShareName, diskName)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("getFileURL(%s,%s,%s,%s) returned with error: %v", accountName, storageEndpointSuffix, fileShareName, diskName, err))
	}
	if fileURL == nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("getFileURL(%s,%s,%s,%s) returned empty fileURL", accountName, storageEndpointSuffix, fileShareName, diskName))
	}

	d.volLockMap.LockEntry(volumeID)
	defer d.volLockMap.UnlockEntry(volumeID)

	if _, err = fileURL.SetMetadata(ctx, azfile.Metadata{metaDataNode: ""}); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("SetMetadata for volume(%s) on node(%s) returned with error: %v", volumeID, nodeID, err))
	}
	klog.V(2).Infof("ControllerUnpublishVolume: volume(%s) detached from node(%s) successfully", volumeID, nodeID)
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

// CreateSnapshot create a snapshot (todo)
func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	klog.V(2).Infof("CreateSnapshot called with request %v", *req)

	sourceVolumeID := req.GetSourceVolumeId()
	snapshotName := req.Name
	if len(snapshotName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot name must be provided")
	}
	if len(sourceVolumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateSnapshot Source Volume ID must be provided")
	}

	exists, item, err := d.snapshotExists(ctx, sourceVolumeID, snapshotName, req.GetSecrets())
	if err != nil {
		if exists {
			return nil, status.Errorf(codes.AlreadyExists, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to check if snapshot(%v) exists: %v", snapshotName, err)
	}
	if exists {
		klog.V(2).Infof("snapshot(%s) already exists", snapshotName)
		tp, err := ptypes.TimestampProto(item.Properties.LastModified)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to covert creation timestamp: %v", err)
		}
		return &csi.CreateSnapshotResponse{
			Snapshot: &csi.Snapshot{
				SizeBytes:      volumehelper.GiBToBytes(int64(item.Properties.Quota)),
				SnapshotId:     sourceVolumeID + "#" + *item.Snapshot,
				SourceVolumeId: sourceVolumeID,
				CreationTime:   tp,
				// Since the snapshot of azurefile has no field of ReadyToUse, here ReadyToUse is always set to true.
				ReadyToUse: true,
			},
		}, nil
	}

	shareURL, err := d.getShareURL(sourceVolumeID, req.GetSecrets())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get share url with (%s): %v", sourceVolumeID, err)
	}

	snapshotShare, err := shareURL.CreateSnapshot(ctx, azfile.Metadata{snapshotNameKey: snapshotName})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create snapshot from(%s) failed with %v, shareURL: %q", sourceVolumeID, err, shareURL)
	}

	klog.V(2).Infof("Created share snapshot: %s", snapshotShare.Snapshot())

	properties, err := shareURL.GetProperties(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get snapshot properties from (%s): %v", snapshotShare.Snapshot(), err)
	}

	tp, err := ptypes.TimestampProto(properties.LastModified())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to covert creation timestamp: %v", err)
	}

	createResp := &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			SizeBytes:      volumehelper.GiBToBytes(int64(properties.Quota())),
			SnapshotId:     sourceVolumeID + "#" + snapshotShare.Snapshot(),
			SourceVolumeId: sourceVolumeID,
			CreationTime:   tp,
			// Since the snapshot of azurefile has no field of ReadyToUse, here ReadyToUse is always set to true.
			ReadyToUse: true,
		},
	}

	return createResp, nil
}

// DeleteSnapshot delete a snapshot (todo)
func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	klog.V(2).Infof("DeleteSnapshot: called with args %+v", *req)
	if len(req.SnapshotId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Snapshot ID must be provided")
	}

	shareURL, err := d.getShareURL(req.SnapshotId, req.GetSecrets())
	if err != nil {
		// According to CSI Driver Sanity Tester, should succeed when an invalid snapshot id is used
		klog.V(4).Infof("failed to get share url with (%s): %v, returning with success", req.SnapshotId, err)
		return &csi.DeleteSnapshotResponse{}, nil
	}

	snapshot, err := getSnapshot(req.SnapshotId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get snapshot name with (%s): %v", req.SnapshotId, err)
	}

	_, err = shareURL.WithSnapshot(snapshot).Delete(ctx, azfile.DeleteSnapshotsOptionNone)
	if err != nil {
		if strings.Contains(err.Error(), "ShareSnapshotNotFound") {
			klog.Warningf("the specify snapshot(%s) was not found", snapshot)
			return &csi.DeleteSnapshotResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to delete snapshot(%s): %v", snapshot, err)
	}

	klog.V(2).Infof("delete snapshot(%s) successfully", snapshot)
	return &csi.DeleteSnapshotResponse{}, nil
}

// ListSnapshots list all snapshots (todo)
func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerExpandVolume controller expand volume
func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.V(2).Infof("ControllerExpandVolume: called with args %+v", *req)
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	capacityBytes := req.GetCapacityRange().GetRequiredBytes()
	if capacityBytes == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capacity range missing in request")
	}
	requestGiB := int32(volumehelper.RoundUpGiB(capacityBytes))
	if err := d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_EXPAND_VOLUME); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid expand volume request: %v", req)
	}

	_, _, _, _, diskName, err := d.getAccountInfo(volumeID, req.GetSecrets(), map[string]string{})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("getAccountInfo(%s) failed with error: %v", volumeID, err))
	}
	if diskName != "" {
		// todo: figure out how to support vhd disk resize
		return nil, status.Error(codes.Unimplemented, fmt.Sprintf("vhd disk volume(%s) is not supported on ControllerExpandVolume", volumeID))
	}

	shareURL, err := d.getShareURL(volumeID, req.GetSecrets())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get share url with (%s): %v, returning with success", volumeID, err)
	}

	if _, err = shareURL.SetQuota(ctx, requestGiB); err != nil {
		return nil, status.Errorf(codes.Internal, "expand volume error: %v", err)
	}

	resp, err := shareURL.GetProperties(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get properties of share(%v): %v", shareURL, err)
	}

	currentQuota := volumehelper.GiBToBytes(int64(resp.Quota()))
	klog.V(2).Infof("ControllerExpandVolume(%s) successfully, currentQuota: %d", volumeID, currentQuota)
	return &csi.ControllerExpandVolumeResponse{CapacityBytes: currentQuota}, nil
}

// getShareURL: sourceVolumeID is the id of source file share, returns a ShareURL of source file share.
// A ShareURL < https://<account>.file.core.windows.net/<fileShareName> > represents a URL to the Azure Storage share allowing you to manipulate its directories and files.
// e.g. The ID of source file share is #fb8fff227be6511e9b24123#createsnapshot-volume-1. Returns https://fb8fff227be6511e9b24123.file.core.windows.net/createsnapshot-volume-1
func (d *Driver) getShareURL(sourceVolumeID string, secrets map[string]string) (azfile.ShareURL, error) {
	serviceURL, fileShareName, err := d.getServiceURL(sourceVolumeID, secrets)
	if err != nil {
		return azfile.ShareURL{}, err
	}

	return serviceURL.NewShareURL(fileShareName), nil
}

func (d *Driver) getServiceURL(sourceVolumeID string, secrets map[string]string) (azfile.ServiceURL, string, error) {
	_, accountName, accountKey, fileShareName, _, err := d.getAccountInfo(sourceVolumeID, secrets, map[string]string{})
	if err != nil {
		return azfile.ServiceURL{}, "", err
	}

	credential, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		klog.Errorf("NewSharedKeyCredential(%s) in CreateSnapshot failed with error: %v", accountName, err)
		return azfile.ServiceURL{}, "", err
	}

	u, err := url.Parse(fmt.Sprintf(serviceURLTemplate, accountName, d.cloud.Environment.StorageEndpointSuffix))
	if err != nil {
		klog.Errorf("parse serviceURLTemplate error: %v", err)
		return azfile.ServiceURL{}, "", err
	}
	if u == nil {
		return azfile.ServiceURL{}, "", fmt.Errorf("url is nil")
	}

	serviceURL := azfile.NewServiceURL(*u, azfile.NewPipeline(credential, azfile.PipelineOptions{}))

	return serviceURL, fileShareName, nil
}

// snapshotExists: sourceVolumeID is the id of source file share, returns the existence of snapshot and its detail info.
// Since `ListSharesSegment` lists all file shares and snapshots, the process of checking existence is divided into two steps.
// 1. Judge if the specify snapshot name already exists.
// 2. If it exists, we should judge if its source file share name equals that we specify.
//    As long as the snapshot already exists, returns true. But when the source is different, an error will be returned.
func (d *Driver) snapshotExists(ctx context.Context, sourceVolumeID, snapshotName string, secrets map[string]string) (bool, azfile.ShareItem, error) {
	serviceURL, fileShareName, err := d.getServiceURL(sourceVolumeID, secrets)
	if err != nil {
		return false, azfile.ShareItem{}, err
	}

	// List share snapshots.
	listSnapshot, err := serviceURL.ListSharesSegment(ctx, azfile.Marker{}, azfile.ListSharesOptions{Detail: azfile.ListSharesDetail{Metadata: true, Snapshots: true}})
	if err != nil {
		return false, azfile.ShareItem{}, err
	}
	for _, share := range listSnapshot.ShareItems {
		if share.Metadata[snapshotNameKey] == snapshotName {
			if share.Name == fileShareName {
				klog.V(2).Infof("found share(%s) snapshot(%s) Metadata(%v)", share.Name, *share.Snapshot, share.Metadata)
				return true, share, nil
			}
			return true, azfile.ShareItem{}, fmt.Errorf("snapshot(%s) already exists, while the current file share name(%s) does not equal to %s, SourceVolumeId(%s)", snapshotName, share.Name, fileShareName, sourceVolumeID)
		}
	}

	return false, azfile.ShareItem{}, nil
}
