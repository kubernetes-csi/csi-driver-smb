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

package smb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	volumeIDRegex            = "^(.*)#(.*)$"
	sourceAndSubDirSeperator = "#"
)

// smbVolume is an internal representation of a volume
// created by the provisioner.
type smbVolume struct {
	// Volume id
	id string
	// Address of the SMB server.
	sourceField string
	// Subdirectory of the SMB server to create volumes under
	subDir string
	// size of volume
	size int64
}

// Ordering of elements in the CSI volume id.
// ID is of the form {server}/{subDir}.
const (
	idsourceField = iota
	idSubDir
	totalIDElements // Always last
)

// CreateVolume only supports static provisioning, no create volume action
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	name := req.GetName()
	if len(name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume name must be provided")
	}

	var volCap *csi.VolumeCapability
	volumeCapabilities := req.GetVolumeCapabilities()
	if len(volumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume Volume capabilities must be provided")
	}
	if len(volumeCapabilities) > 0 {
		volCap = req.GetVolumeCapabilities()[0]
	}

	reqCapacity := req.GetCapacityRange().GetRequiredBytes()
	parameters := req.GetParameters()
	// Validate parameters (case-insensitive).
	for k := range parameters {
		switch strings.ToLower(k) {
		case paramSource:
			// no op
		default:
			return nil, fmt.Errorf("invalid parameter %s in storage class", k)
		}
	}

	smbVol, err := d.newSMBVolume(name, reqCapacity, parameters)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets := req.GetSecrets()
	if len(secrets) > 0 {
		// Mount smb base share so we can create a subdirectory
		if err := d.internalMount(ctx, smbVol, volCap, secrets); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount smb server: %v", err.Error())
		}
		defer func() {
			if err = d.internalUnmount(ctx, smbVol); err != nil {
				klog.Warningf("failed to unmount smb server: %v", err.Error())
			}
		}()
		// Create subdirectory under base-dir
		// TODO: revisit permissions
		internalVolumePath := d.getInternalVolumePath(smbVol)
		if err = os.Mkdir(internalVolumePath, 0777); err != nil && !os.IsExist(err) {
			return nil, status.Errorf(codes.Internal, "failed to make subdirectory: %v", err.Error())
		}
		parameters[sourceField] = parameters[sourceField] + "/" + smbVol.subDir
	} else {
		klog.Infof("CreateVolume(%s) does not provide secrets", name)
	}
	return &csi.CreateVolumeResponse{Volume: d.smbVolToCSI(smbVol, parameters)}, nil
}

// DeleteVolume only supports static provisioning, no delete volume action
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume id is empty")
	}
	smbVol, err := getSmbVolFromID(volumeID)
	if err != nil {
		// An invalid ID should be treated as doesn't exist
		klog.Warningf("failed to get smb volume for volume id %v deletion: %v", volumeID, err)
		return &csi.DeleteVolumeResponse{}, nil
	}

	secrets := req.GetSecrets()
	if len(secrets) > 0 {
		// Mount smb base share so we can delete the subdirectory
		if err = d.internalMount(ctx, smbVol, nil, secrets); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount smb server: %v", err.Error())
		}
		defer func() {
			if err = d.internalUnmount(ctx, smbVol); err != nil {
				klog.Warningf("failed to unmount smb server: %v", err.Error())
			}
		}()

		// Delete subdirectory under base-dir
		internalVolumePath := d.getInternalVolumePath(smbVol)
		klog.V(2).Infof("Removing subdirectory at %v", internalVolumePath)
		if err = os.RemoveAll(internalVolumePath); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to delete subdirectory: %v", err.Error())
		}
	} else {
		klog.Infof("DeleteVolume(%s) does not provide secrets", volumeID)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerGetVolume get volume
func (d *Driver) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns the capabilities of the Controller plugin
func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: d.Cap,
	}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}

	// supports all AccessModes, no need to check capabilities here
	return &csi.ValidateVolumeCapabilitiesResponse{Message: ""}, nil
}

// GetCapacity returns the capacity of the total available storage pool
func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes return all available volumes
func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerExpandVolume expand volume
func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// Given a smbVolume, return a CSI volume id
func (d *Driver) getVolumeIDFromSmbVol(vol *smbVolume) string {
	idElements := make([]string, totalIDElements)
	idElements[idsourceField] = strings.Trim(vol.sourceField, "/")
	idElements[idSubDir] = strings.Trim(vol.subDir, "/")
	return strings.Join(idElements, sourceAndSubDirSeperator)
}

// Get working directory for CreateVolume and DeleteVolume
func (d *Driver) getInternalMountPath(vol *smbVolume) string {
	// use default if empty
	if d.workingMountDir == "" {
		d.workingMountDir = "/tmp"
	}
	return filepath.Join(d.workingMountDir, vol.subDir)
}

// Mount smb server at base-dir
func (d *Driver) internalMount(ctx context.Context, vol *smbVolume, volCap *csi.VolumeCapability, secrets map[string]string) error {
	stagingPath := d.getInternalMountPath(vol)

	if volCap == nil {
		volCap = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
		}
	}

	klog.V(4).Infof("internally mounting %v at %v", sourceField, stagingPath)
	_, err := d.NodeStageVolume(ctx, &csi.NodeStageVolumeRequest{
		StagingTargetPath: stagingPath,
		VolumeContext: map[string]string{
			sourceField: vol.sourceField,
		},
		VolumeCapability: volCap,
		VolumeId:         vol.id,
		Secrets:          secrets,
	})
	return err
}

// Unmount smb server at base-dir
func (d *Driver) internalUnmount(ctx context.Context, vol *smbVolume) error {
	targetPath := d.getInternalMountPath(vol)

	// Unmount smb server at base-dir
	klog.V(4).Infof("internally unmounting %v", targetPath)
	_, err := d.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          vol.id,
		StagingTargetPath: d.getInternalMountPath(vol),
	})
	return err
}

// Convert VolumeCreate parameters to an smbVolume
func (d *Driver) newSMBVolume(name string, size int64, params map[string]string) (*smbVolume, error) {
	var sourceField string

	// Validate parameters (case-insensitive).
	for k, v := range params {
		switch strings.ToLower(k) {
		case paramSource:
			sourceField = v
		}
	}

	// Validate required parameters
	if sourceField == "" {
		return nil, fmt.Errorf("%v is a required parameter", paramSource)
	}

	vol := &smbVolume{
		sourceField: sourceField,
		subDir:      name,
		size:        size,
	}
	vol.id = d.getVolumeIDFromSmbVol(vol)

	return vol, nil
}

// Get internal path where the volume is created
// The reason why the internal path is "workingDir/subDir/subDir" is because:
//   * the semantic is actually "workingDir/volId/subDir" and volId == subDir.
//   * we need a mount directory per volId because you can have multiple
//     CreateVolume calls in parallel and they may use the same underlying share.
//     Instead of refcounting how many CreateVolume calls are using the same
//     share, it's simpler to just do a mount per request.
func (d *Driver) getInternalVolumePath(vol *smbVolume) string {
	return filepath.Join(d.getInternalMountPath(vol), vol.subDir)
}

// Convert into smbVolume into a csi.Volume
func (d *Driver) smbVolToCSI(vol *smbVolume, parameters map[string]string) *csi.Volume {
	return &csi.Volume{
		CapacityBytes: 0, // by setting it to zero, Provisioner will use PVC requested size as PV size
		VolumeId:      vol.id,
		VolumeContext: parameters,
	}
}

// Given a CSI volume id, return a smbVolume
// a sample volume Id: smb-server.default.svc.cluster.local/share/pvc-4729891a-f57e-4982-9c60-e9884af1be2f
func getSmbVolFromID(id string) (*smbVolume, error) {
	volRegex := regexp.MustCompile(volumeIDRegex)
	tokens := volRegex.FindStringSubmatch(id)
	if tokens == nil || len(tokens) != 3 {
		return nil, fmt.Errorf("Could not split %q into server and subDir", id)
	}
	source := "//" + tokens[1]
	return &smbVolume{
		id:          id,
		sourceField: source,
		subDir:      tokens[2],
	}, nil
}
