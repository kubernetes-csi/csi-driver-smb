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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	azcache "sigs.k8s.io/cloud-provider-azure/pkg/cache"
)

const (
	separator = "#"
)

// smbVolume is an internal representation of a volume
// created by the provisioner.
type smbVolume struct {
	// Volume id
	id string
	// Address of the SMB server.
	source string
	// Subdirectory of the SMB server to create volumes under
	subDir string
	// size of volume
	size int64
	// pv name when subDir is not empty
	uuid string
	// on delete action
	onDelete string
}

// Ordering of elements in the CSI volume id.
// ID is of the form {server}/{subDir}.
const (
	idSource = iota
	idSubDir
	idUUID
	idOnDelete
	totalIDElements // Always last
)

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	name := req.GetName()
	if len(name) == 0 {
		return nil, status.Error(codes.InvalidArgument, "CreateVolume name must be provided")
	}

	volumeCapabilities := req.GetVolumeCapabilities()
	if err := isValidVolumeCapabilities(volumeCapabilities); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	reqCapacity := req.GetCapacityRange().GetRequiredBytes()
	parameters := req.GetParameters()
	if parameters == nil {
		parameters = make(map[string]string)
	}
	smbVol, err := newSMBVolume(name, reqCapacity, parameters, d.defaultOnDeletePolicy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets := req.GetSecrets()
	createSubDir := len(secrets) > 0
	if len(smbVol.uuid) > 0 {
		klog.V(2).Infof("create subdirectory(%s) if not exists", smbVol.subDir)
		createSubDir = true
	}

	volCap := volumeCapabilities[0]
	if volCap.GetMount() != nil {
		options := volCap.GetMount().GetMountFlags()
		if !createSubDir && hasGuestMountOptions(options) {
			klog.V(2).Infof("guest mount option(%v) is provided, create subdirectory", options)
			createSubDir = true
		}
		// set default file/dir mode
		volCap.GetMount().MountFlags = appendMountOptions(options, map[string]string{
			fileMode: defaultFileMode,
			dirMode:  defaultDirMode,
		})
	}

	if acquired := d.volumeLocks.TryAcquire(name); !acquired {
		return nil, status.Errorf(codes.Aborted, volumeOperationAlreadyExistsFmt, name)
	}
	defer d.volumeLocks.Release(name)

	if createSubDir {
		// Mount smb base share so we can create a subdirectory
		if err := d.internalMount(ctx, smbVol, volCap, secrets); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount smb server: %v", err)
		}
		defer func() {
			if err = d.internalUnmount(ctx, smbVol); err != nil {
				klog.Warningf("failed to unmount smb server: %v", err)
			}
		}()
		// Create subdirectory under base-dir
		// TODO: revisit permissions
		internalVolumePath := getInternalVolumePath(d.workingMountDir, smbVol)
		if err = os.MkdirAll(internalVolumePath, 0777); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to make subdirectory: %v", err)
		}

		if req.GetVolumeContentSource() != nil {
			if err := d.copyVolume(ctx, req, smbVol); err != nil {
				return nil, err
			}
		}

		setKeyValueInMap(parameters, subDirField, smbVol.subDir)
	} else {
		klog.V(2).Infof("CreateVolume(%s) does not create subdirectory", name)

		if req.GetVolumeContentSource() != nil {
			if err := d.copyVolume(ctx, req, smbVol); err != nil {
				return nil, err
			}
		}
	}
	return &csi.CreateVolumeResponse{Volume: d.smbVolToCSI(smbVol, req, parameters)}, nil
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

	if acquired := d.volumeLocks.TryAcquire(volumeID); !acquired {
		return nil, status.Errorf(codes.Aborted, volumeOperationAlreadyExistsFmt, volumeID)
	}
	defer d.volumeLocks.Release(volumeID)

	secrets := req.GetSecrets()
	mountOptions := getMountOptions(secrets)
	if mountOptions != "" {
		klog.V(2).Infof("DeleteVolume: found mountOptions(%v) for volume(%s)", mountOptions, volumeID)
	}
	// set default file/dir mode
	volCap := &csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				MountFlags: appendMountOptions([]string{mountOptions},
					map[string]string{
						fileMode: defaultFileMode,
						dirMode:  defaultDirMode,
					}),
			},
		},
	}

	if smbVol.onDelete == "" {
		smbVol.onDelete = d.defaultOnDeletePolicy
	}

	if len(req.GetSecrets()) > 0 && !strings.EqualFold(smbVol.onDelete, retain) {
		klog.V(2).Infof("begin to delete or archive subdirectory since secret is provided")
		// check whether volumeID is in the cache
		cache, err := d.volDeletionCache.Get(volumeID, azcache.CacheReadTypeDefault)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
		if cache != nil {
			klog.V(2).Infof("DeleteVolume: volume %s is already deleted", volumeID)
			return &csi.DeleteVolumeResponse{}, nil
		}

		// mount smb base share so we can delete or archive the subdirectory
		if err = d.internalMount(ctx, smbVol, volCap, secrets); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to mount smb server: %v", err)
		}
		defer func() {
			if err = d.internalUnmount(ctx, smbVol); err != nil {
				klog.Warningf("failed to unmount smb server: %v", err)
			}
		}()

		internalVolumePath := getInternalVolumePath(d.workingMountDir, smbVol)
		if strings.EqualFold(smbVol.onDelete, archive) {
			archivedInternalVolumePath := filepath.Join(getInternalMountPath(d.workingMountDir, smbVol), "archived-"+smbVol.subDir)

			if strings.Contains(smbVol.subDir, "/") {
				parentDir := filepath.Dir(archivedInternalVolumePath)
				klog.V(2).Infof("DeleteVolume: subdirectory(%s) contains '/', make sure the parent directory(%s) exists", smbVol.subDir, parentDir)
				if err = os.MkdirAll(parentDir, 0777); err != nil {
					return nil, status.Errorf(codes.Internal, "create parent directory(%s) of %s failed with %v", parentDir, archivedInternalVolumePath, err)
				}
			}

			// archive subdirectory under base-dir. Remove stale archived copy if exists.
			klog.V(2).Infof("archiving subdirectory %s --> %s", internalVolumePath, archivedInternalVolumePath)
			if d.removeArchivedVolumePath {
				klog.V(2).Infof("removing archived subdirectory at %v", archivedInternalVolumePath)
				if err = os.RemoveAll(archivedInternalVolumePath); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to delete archived subdirectory %s: %v", archivedInternalVolumePath, err)
				}
				klog.V(2).Infof("removed archived subdirectory at %v", archivedInternalVolumePath)
			}
			if err = os.Rename(internalVolumePath, archivedInternalVolumePath); err != nil {
				return nil, status.Errorf(codes.Internal, "archive subdirectory(%s, %s) failed with %v", internalVolumePath, archivedInternalVolumePath, err)
			}
		} else {
			if _, err := os.Lstat(internalVolumePath); err == nil {
				if err2 := filepath.WalkDir(internalVolumePath, func(path string, _ fs.DirEntry, _ error) error {
					return os.Chmod(path, 0777)
				}); err2 != nil {
					klog.Errorf("failed to chmod subdirectory: %v", err2)
				}
			}

			rootDir := getRootDir(smbVol.subDir)
			if rootDir != "" {
				rootDir = filepath.Join(getInternalMountPath(d.workingMountDir, smbVol), rootDir)
			} else {
				rootDir = internalVolumePath
			}

			klog.V(2).Infof("removing subdirectory at %v on internalVolumePath %s", rootDir, internalVolumePath)
			if err = os.RemoveAll(internalVolumePath); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to delete subdirectory: %v", err)
			}
		}
	} else {
		klog.V(2).Infof("DeleteVolume(%s) does not delete subdirectory", volumeID)
	}

	d.volDeletionCache.Set(volumeID, "")
	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerGetVolume get volume
func (d *Driver) ControllerGetVolume(context.Context, *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerPublishVolume(_ context.Context, _ *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns the capabilities of the Controller plugin
func (d *Driver) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: d.Cap,
	}, nil
}

func (d *Driver) ValidateVolumeCapabilities(_ context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if err := isValidVolumeCapabilities(req.GetVolumeCapabilities()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		},
		Message: "",
	}, nil
}

// GetCapacity returns the capacity of the total available storage pool
func (d *Driver) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes return all available volumes
func (d *Driver) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerExpandVolume expand volume
func (d *Driver) ControllerExpandVolume(_ context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if req.GetCapacityRange() == nil {
		return nil, status.Error(codes.InvalidArgument, "Capacity Range missing in request")
	}

	volSizeBytes := int64(req.GetCapacityRange().GetRequiredBytes())
	klog.V(2).Infof("ControllerExpandVolume(%s) successfully, currentQuota: %d bytes", req.VolumeId, volSizeBytes)

	return &csi.ControllerExpandVolumeResponse{CapacityBytes: req.GetCapacityRange().GetRequiredBytes()}, nil
}

func (d *Driver) CreateSnapshot(_ context.Context, _ *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) DeleteSnapshot(_ context.Context, _ *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (d *Driver) ListSnapshots(_ context.Context, _ *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// Mount smb server at base-dir
func (d *Driver) internalMount(ctx context.Context, vol *smbVolume, volCap *csi.VolumeCapability, secrets map[string]string) error {
	stagingPath := getInternalMountPath(d.workingMountDir, vol)

	if volCap == nil {
		volCap = &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
		}
	}

	klog.V(4).Infof("internally mounting %v at %v", vol.source, stagingPath)
	_, err := d.NodeStageVolume(ctx, &csi.NodeStageVolumeRequest{
		StagingTargetPath: stagingPath,
		VolumeContext: map[string]string{
			sourceField: vol.source,
		},
		VolumeCapability: volCap,
		VolumeId:         vol.id,
		Secrets:          secrets,
	})
	return err
}

// Unmount smb server at base-dir
func (d *Driver) internalUnmount(ctx context.Context, vol *smbVolume) error {
	targetPath := getInternalMountPath(d.workingMountDir, vol)

	// Unmount smb server at base-dir
	klog.V(4).Infof("internally unmounting %v", targetPath)
	_, err := d.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          vol.id,
		StagingTargetPath: targetPath,
	})
	return err
}

// copyFromVolume create a copied volume from a volume
func (d *Driver) copyFromVolume(ctx context.Context, req *csi.CreateVolumeRequest, dstVol *smbVolume) error {
	srcVol, err := getSmbVolFromID(req.GetVolumeContentSource().GetVolume().GetVolumeId())
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}
	// Note that the source path must include trailing '/.', can't use 'filepath.Join()' as it performs path cleaning
	srcPath := fmt.Sprintf("%v/.", getInternalVolumePath(d.workingMountDir, srcVol))
	dstPath := getInternalVolumePath(d.workingMountDir, dstVol)
	klog.V(2).Infof("copy volume from volume %v -> %v", srcPath, dstPath)

	var volCap *csi.VolumeCapability
	if len(req.GetVolumeCapabilities()) > 0 {
		volCap = req.GetVolumeCapabilities()[0]
	}

	secrets := req.GetSecrets()
	if err = d.internalMount(ctx, srcVol, volCap, secrets); err != nil {
		return status.Errorf(codes.Internal, "failed to mount src nfs server: %v", err)
	}
	defer func() {
		if err = d.internalUnmount(ctx, srcVol); err != nil {
			klog.Warningf("failed to unmount nfs server: %v", err)
		}
	}()
	if err = d.internalMount(ctx, dstVol, volCap, secrets); err != nil {
		return status.Errorf(codes.Internal, "failed to mount dst nfs server: %v", err)
	}
	defer func() {
		if err = d.internalUnmount(ctx, dstVol); err != nil {
			klog.Warningf("failed to unmount dst nfs server: %v", err)
		}
	}()

	// recursive 'cp' with '-a' to handle symlinks
	out, err := exec.Command("cp", "-a", srcPath, dstPath).CombinedOutput()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to copy volume %v: %v", err, string(out))
	}
	klog.V(2).Infof("copied %s -> %s", srcPath, dstPath)
	return nil
}

func (d *Driver) copyVolume(ctx context.Context, req *csi.CreateVolumeRequest, vol *smbVolume) error {
	vs := req.VolumeContentSource
	switch vs.Type.(type) {
	case *csi.VolumeContentSource_Snapshot:
		return status.Errorf(codes.InvalidArgument, "copy volume from volumeSnapshot is not supported")
	case *csi.VolumeContentSource_Volume:
		return d.copyFromVolume(ctx, req, vol)
	default:
		return status.Errorf(codes.InvalidArgument, "%v is not a proper volume source", vs)
	}
}

// Given a smbVolume, return a CSI volume id
func getVolumeIDFromSmbVol(vol *smbVolume) string {
	idElements := make([]string, totalIDElements)
	idElements[idSource] = strings.Trim(vol.source, "/")
	idElements[idSubDir] = strings.Trim(vol.subDir, "/")
	idElements[idUUID] = vol.uuid
	if strings.EqualFold(vol.onDelete, retain) || strings.EqualFold(vol.onDelete, archive) {
		idElements[idOnDelete] = vol.onDelete
	}
	return strings.Join(idElements, separator)
}

// getInternalMountPath: get working directory for CreateVolume and DeleteVolume
func getInternalMountPath(workingMountDir string, vol *smbVolume) string {
	if vol == nil {
		return ""
	}
	mountDir := vol.uuid
	if vol.uuid == "" {
		mountDir = vol.subDir
	}
	return filepath.Join(workingMountDir, mountDir)
}

// Convert VolumeCreate parameters to an smbVolume
func newSMBVolume(name string, size int64, params map[string]string, defaultOnDeletePolicy string) (*smbVolume, error) {
	var source, subDir, onDelete string
	subDirReplaceMap := map[string]string{}

	// validate parameters (case-insensitive).
	for k, v := range params {
		switch strings.ToLower(k) {
		case sourceField:
			source = v
		case subDirField:
			subDir = v
		case paramOnDelete:
			onDelete = v
		case pvcNamespaceKey:
			subDirReplaceMap[pvcNamespaceMetadata] = v
		case pvcNameKey:
			subDirReplaceMap[pvcNameMetadata] = v
		case pvNameKey:
			subDirReplaceMap[pvNameMetadata] = v
		default:
			return nil, fmt.Errorf("invalid parameter %s in storage class", k)
		}
	}

	if source == "" {
		return nil, fmt.Errorf("%v is a required parameter", sourceField)
	}

	vol := &smbVolume{
		source: source,
		size:   size,
	}
	if subDir == "" {
		// use pv name by default if not specified
		vol.subDir = name
	} else {
		// replace pv/pvc name namespace metadata in subDir
		vol.subDir = replaceWithMap(subDir, subDirReplaceMap)
		// make volume id unique if subDir is provided
		vol.uuid = name
	}

	if err := validateOnDeleteValue(onDelete); err != nil {
		return nil, err
	}

	vol.onDelete = defaultOnDeletePolicy
	if onDelete != "" {
		vol.onDelete = onDelete
	}

	vol.id = getVolumeIDFromSmbVol(vol)
	return vol, nil
}

// Get internal path where the volume is created
// The reason why the internal path is "workingDir/subDir/subDir" is because:
//   - the semantic is actually "workingDir/volId/subDir" and volId == subDir.
//   - we need a mount directory per volId because you can have multiple
//     CreateVolume calls in parallel and they may use the same underlying share.
//     Instead of refcounting how many CreateVolume calls are using the same
//     share, it's simpler to just do a mount per request.
func getInternalVolumePath(workingMountDir string, vol *smbVolume) string {
	return filepath.Join(getInternalMountPath(workingMountDir, vol), vol.subDir)
}

// Convert into smbVolume into a csi.Volume
func (d *Driver) smbVolToCSI(vol *smbVolume, req *csi.CreateVolumeRequest, parameters map[string]string) *csi.Volume {
	return &csi.Volume{
		CapacityBytes: 0, // by setting it to zero, Provisioner will use PVC requested size as PV size
		VolumeId:      vol.id,
		VolumeContext: parameters,
		ContentSource: req.GetVolumeContentSource(),
	}
}

// Given a CSI volume id, return a smbVolume
// sample volume Id:
//
//	smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f
//	smb-server.default.svc.cluster.local/share#subdir#pvc-4729891a-f57e-4982-9c60-e9884af1be2f
func getSmbVolFromID(id string) (*smbVolume, error) {
	segments := strings.Split(id, separator)
	if len(segments) < 2 {
		return nil, fmt.Errorf("could not split %q into server and subDir", id)
	}
	source := segments[0]
	if !strings.HasPrefix(segments[0], "//") {
		source = "//" + source
	}
	vol := &smbVolume{
		id:     id,
		source: source,
		subDir: segments[1],
	}
	if len(segments) >= 3 {
		vol.uuid = segments[2]
	}
	if len(segments) >= 4 {
		vol.onDelete = segments[3]
	}
	return vol, nil
}

// isValidVolumeCapabilities validates the given VolumeCapability array is valid
func isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) error {
	if len(volCaps) == 0 {
		return fmt.Errorf("volume capabilities missing in request")
	}
	for _, c := range volCaps {
		if c.GetBlock() != nil {
			return fmt.Errorf("block volume capability not supported")
		}
	}
	return nil
}
