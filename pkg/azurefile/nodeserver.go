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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"
)

// NodePublishVolume mount the volume from staging to target path
func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	source := req.GetStagingTargetPath()
	if len(source) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	target := req.GetTargetPath()
	if len(target) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path not provided")
	}

	mountOptions := []string{"bind"}
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	mnt, err := d.ensureMountPoint(target)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not mount target %q: %v", target, err)
	}
	if mnt {
		klog.V(2).Infof("NodePublishVolume: %s is already mounted", target)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	klog.V(2).Infof("NodePublishVolume: mounting %s at %s with mountOptions: %v", source, target, mountOptions)
	if err := d.mounter.Mount(source, target, "", mountOptions); err != nil {
		if removeErr := os.Remove(target); removeErr != nil {
			return nil, status.Errorf(codes.Internal, "Could not remove mount target %q: %v", target, removeErr)
		}
		return nil, status.Errorf(codes.Internal, "Could not mount %q at %q: %v", source, target, err)
	}
	klog.V(2).Infof("NodePublishVolume: mount %s at %s successfully", source, target)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume from the target path
func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(2).Infof("NodeUnPublishVolume: called with args %+v", *req)
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	targetPath := req.GetTargetPath()
	volumeID := req.GetVolumeId()

	klog.V(2).Infof("NodeUnpublishVolume: unmounting volume %s on %s", volumeID, targetPath)
	err := mount.CleanupMountPoint(targetPath, d.mounter, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmount target %q: %v", targetPath, err)
	}
	klog.V(2).Infof("NodeUnpublishVolume: unmount volume %s on %s successfully", volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeStageVolume mount the volume to a staging path
func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}
	volumeCapability := req.GetVolumeCapability()
	if volumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not provided")
	}

	volumeID := req.GetVolumeId()
	attrib := req.GetVolumeContext()
	mountFlags := req.GetVolumeCapability().GetMount().GetMountFlags()

	var accountName, accountKey, fileShareName string
	var err error

	secrets := req.GetSecrets()
	if len(secrets) == 0 {
		var resourceGroupName string
		resourceGroupName, accountName, fileShareName, err = getFileShareInfo(volumeID)
		if err != nil {
			return nil, err
		}

		if resourceGroupName == "" {
			resourceGroupName = d.cloud.ResourceGroup
		}

		accountKey, err = d.cloud.GetStorageAccesskey(accountName, resourceGroupName)
		if err != nil {
			return nil, fmt.Errorf("no key for storage account(%s) under resource group(%s), err %v", accountName, resourceGroupName, err)
		}
	} else {
		for k, v := range attrib {
			switch strings.ToLower(k) {
			case "sharename":
				fileShareName = v
			}
		}
		if fileShareName == "" {
			return nil, fmt.Errorf("could not find sharename from attributes(%v)", attrib)
		}

		accountName, accountKey, err = getStorageAccount(secrets)
		if err != nil {
			return nil, err
		}
	}
	// don't respect fsType from req.GetVolumeCapability().GetMount().GetFsType()
	// since it's ext4 by default on Linux
	var diskName, fsType string
	for k, v := range attrib {
		switch strings.ToLower(k) {
		case fsTypeField:
			fsType = v
		case diskNameField:
			diskName = v
		}
	}

	var mountOptions []string
	osSeparator := string(os.PathSeparator)
	source := fmt.Sprintf("%s%s%s.file.%s%s%s", osSeparator, osSeparator, accountName, d.cloud.Environment.StorageEndpointSuffix, osSeparator, fileShareName)

	cifsMountPath := targetPath
	cifsMountFlags := mountFlags
	isDiskMount := (fsType != "" && fsType != cifs)
	if isDiskMount {
		if diskName == "" {
			return nil, status.Errorf(codes.Internal, "diskname could not be empty, targetPath: %s", targetPath)
		}
		cifsMountFlags = []string{"dir_mode=0777,file_mode=0777,cache=strict,actimeo=30"}
		cifsMountPath = filepath.Join(filepath.Dir(targetPath), proxyMount)
	}

	if runtime.GOOS == "windows" {
		mountOptions = []string{fmt.Sprintf("AZURE\\%s", accountName), accountKey}
	} else {
		if err := os.MkdirAll(targetPath, 0750); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("MkdirAll %s failed with error: %v", targetPath, err))
		}
		// parameters suggested by https://azure.microsoft.com/en-us/documentation/articles/storage-how-to-use-files-linux/
		options := []string{fmt.Sprintf("username=%s,password=%s", accountName, accountKey)}
		mountOptions = util.JoinMountOptions(cifsMountFlags, options)
		mountOptions = appendDefaultMountOptions(mountOptions)
	}

	klog.V(2).Infof("cifsMountPath(%v) fstype(%v) volumeID(%v) context(%v) mountflags(%v) mountOptions(%v)",
		cifsMountPath, fsType, volumeID, attrib, mountFlags, mountOptions)

	isDirMounted, err := d.ensureMountPoint(cifsMountPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not mount target %q: %v", cifsMountPath, err)
	}

	if !isDirMounted {
		mountComplete := false
		err = wait.Poll(5*time.Second, 10*time.Minute, func() (bool, error) {
			err := d.mounter.Mount(source, cifsMountPath, cifs, mountOptions)
			mountComplete = true
			return true, err
		})
		if !mountComplete {
			return nil, status.Error(codes.Internal, fmt.Sprintf("volume(%s) mount %q on %q failed with timeout(10m)", volumeID, source, cifsMountPath))
		}
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("volume(%s) mount %q on %q failed with %v", volumeID, source, cifsMountPath, err))
		}
		klog.V(2).Infof("volume(%s) mount %q on %q succeeded", volumeID, source, cifsMountPath)
	}

	if isDiskMount {
		diskPath := filepath.Join(cifsMountPath, diskName)
		// todo: add lock for loop device
		options := util.JoinMountOptions(mountFlags, []string{"loop"})
		// FormatAndMount will format only if needed
		klog.V(2).Infof("NodeStageVolume: formatting %s and mounting at %s with mount options(%s)", targetPath, diskPath, options)
		if err := d.mounter.FormatAndMount(diskPath, targetPath, fsType, options); err != nil {
			msg := fmt.Sprintf("could not format %q and mount it at %q", targetPath, diskPath)
			return nil, status.Error(codes.Internal, msg)
		}
		klog.V(2).Infof("NodeStageVolume: format %s and mounting at %s successfully.", targetPath, diskPath)
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume unmount the volume from the staging path
func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.V(2).Infof("NodeUnstageVolume: called with args %+v", *req)
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Staging target not provided")
	}

	klog.V(2).Infof("NodeUnstageVolume: CleanupMountPoint %s", stagingTargetPath)
	if err := mount.CleanupMountPoint(stagingTargetPath, d.mounter, false); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmount staing target %q: %v", stagingTargetPath, err)
	}

	targetPath := filepath.Join(filepath.Dir(stagingTargetPath), proxyMount)
	klog.V(2).Infof("NodeUnstageVolume: CleanupMountPoint %s", targetPath)
	if err := mount.CleanupMountPoint(targetPath, d.mounter, false); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmount staing target %q: %v", targetPath, err)
	}
	klog.V(2).Infof("NodeUnstageVolume: unmount %s successfully", stagingTargetPath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodeGetCapabilities return the capabilities of the Node plugin
func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(2).Infof("Using default NodeGetCapabilities")

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: d.NSCap,
	}, nil
}

// NodeGetInfo return info of the node on which this plugin is running
func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(5).Infof("Using default NodeGetInfo")

	return &csi.NodeGetInfoResponse{
		NodeId: d.NodeID,
	}, nil
}

// NodeGetVolumeStats get volume stats
func (d *Driver) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeExpandVolume node expand volume
// N/A for azure file
func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ensureMountPoint: create mount point if not exists
// return <true, nil> if it's already a mounted point otherwise return <false, nil>
func (d *Driver) ensureMountPoint(target string) (bool, error) {
	notMnt, err := d.mounter.IsLikelyNotMountPoint(target)
	if err != nil && !os.IsNotExist(err) {
		if IsCorruptedDir(target) {
			notMnt = false
			klog.Warningf("detected corrupted mount for targetPath [%s]", target)
		} else {
			return !notMnt, err
		}
	}

	if !notMnt {
		// testing original mount point, make sure the mount link is valid
		_, err := ioutil.ReadDir(target)
		if err == nil {
			klog.V(2).Infof("already mounted to target %s", target)
			return !notMnt, nil
		}
		// mount link is invalid, now unmount and remount later
		klog.Warningf("ReadDir %s failed with %v, unmount this directory", target, err)
		if err := d.mounter.Unmount(target); err != nil {
			klog.Errorf("Unmount directory %s failed with %v", target, err)
			return !notMnt, err
		}
		notMnt = true
		return !notMnt, err
	}

	if err := d.mounter.MakeDir(target); err != nil {
		klog.Errorf("MakeDir failed on target: %s (%v)", target, err)
		return !notMnt, err
	}

	return false, nil
}
