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
	"runtime"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"

	"k8s.io/kubernetes/pkg/volume/util"
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

	targetPath := req.GetTargetPath()
	notMnt, err := d.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if !notMnt {
		// testing original mount point, make sure the mount link is valid
		if _, err := ioutil.ReadDir(targetPath); err == nil {
			glog.V(2).Infof("azureFile - already mounted to target %s", targetPath)
			return &csi.NodePublishVolumeResponse{}, nil
		}
		// todo: mount link is invalid, now unmount and remount later (built-in functionality)
		glog.Warningf("azureFile - ReadDir %s failed with %v, unmount this directory", targetPath, err)
		if err := d.mounter.Unmount(targetPath); err != nil {
			glog.Errorf("azureFile - Unmount directory %s failed with %v", targetPath, err)
			return nil, err
		}
		notMnt = true
	}

	fsType := req.GetVolumeCapability().GetMount().GetFsType()

	readOnly := req.GetReadonly()
	volumeID := req.GetVolumeId()
	attrib := req.GetVolumeAttributes()
	mountFlags := req.GetVolumeCapability().GetMount().GetMountFlags()

	glog.V(2).Infof("target %v\nfstype %v\n\nreadonly %v\nvolumeId %v\nattributes %v\nmountflags %v\n",
		targetPath, fsType, readOnly, volumeID, attrib, mountFlags)

	var accountName, accountKey, fileShareName string

	secrets := req.GetNodePublishSecrets()
	if len(secrets) == 0 {
		var resourceGroupName string
		resourceGroupName, accountName, fileShareName, err = getFileShareInfo(volumeID)
		if err != nil {
			return nil, err
		}

		if resourceGroupName == "" {
			resourceGroupName = d.cloud.ResourceGroup
		}

		accountKey, err = GetStorageAccesskey(d.cloud, accountName, resourceGroupName)
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

	mountOptions := []string{}
	source := ""
	osSeparator := string(os.PathSeparator)
	source = fmt.Sprintf("%s%s%s.file.%s%s%s", osSeparator, osSeparator, accountName, d.cloud.Environment.StorageEndpointSuffix, osSeparator, fileShareName)

	if runtime.GOOS == "windows" {
		mountOptions = []string{fmt.Sprintf("AZURE\\%s", accountName), accountKey}
	} else {
		if err := os.MkdirAll(targetPath, 0700); err != nil {
			return nil, err
		}
		// parameters suggested by https://azure.microsoft.com/en-us/documentation/articles/storage-how-to-use-files-linux/
		options := []string{fmt.Sprintf("username=%s,password=%s", accountName, accountKey)}
		if readOnly {
			options = append(options, "ro")
		}
		mountOptions = util.JoinMountOptions(mountFlags, options)
		mountOptions = appendDefaultMountOptions(mountOptions)
	}

	err = d.mounter.Mount(source, targetPath, "cifs", mountOptions)
	if err != nil {
		notMnt, mntErr := d.mounter.IsLikelyNotMountPoint(targetPath)
		if mntErr != nil {
			glog.Errorf("IsLikelyNotMountPoint check failed: %v", mntErr)
			return nil, err
		}
		if !notMnt {
			if mntErr = d.mounter.Unmount(targetPath); mntErr != nil {
				glog.Errorf("Failed to unmount: %v", mntErr)
				return nil, err
			}
			notMnt, mntErr := d.mounter.IsLikelyNotMountPoint(targetPath)
			if mntErr != nil {
				glog.Errorf("IsLikelyNotMountPoint check failed: %v", mntErr)
				return nil, err
			}
			if !notMnt {
				// This is very odd, we don't expect it.  We'll try again next sync loop.
				glog.Errorf("%s is still mounted, despite call to unmount().  Will try again next sync loop.", targetPath)
				return nil, err
			}
		}
		os.Remove(targetPath)
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume from the target path
func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}
	targetPath := req.GetTargetPath()
	volumeID := req.GetVolumeId()

	// Unmounting the image
	err := d.mounter.Unmount(req.GetTargetPath())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	glog.V(4).Infof("azurefile: volume %s/%s has been unmounted.", targetPath, volumeID)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeStageVolume mount the volume to a staging path
// todo: we may implement this for azure file
// The reason that mounting is a two step operation is
// because Kubernetes allows you to use a single volume by multiple pods.
// This is allowed when the storage system supports it or if all pods run on the same node.
func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetStagingTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume unmount the volume from the staging path
// todo: we may implement this for azure file
// The reason that mounting is a two step operation is
// because Kubernetes allows you to use a single volume by multiple pods.
// This is allowed when the storage system supports it or if all pods run on the same node.
func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetStagingTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

// NodeGetCapabilities return the capabilities of the Node plugin
func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	glog.V(2).Infof("Using default NodeGetCapabilities")

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: d.NSCap,
	}, nil
}

// NodeGetId return a unique ID of the node on which this plugin is running
func (d *Driver) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	glog.V(5).Infof("Using default NodeGetId")

	return &csi.NodeGetIdResponse{
		NodeId: d.NodeID,
	}, nil
}

// NodeGetInfo return info of the node on which this plugin is running
func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	glog.V(5).Infof("Using default NodeGetInfo")

	return &csi.NodeGetInfoResponse{
		NodeId: d.NodeID,
	}, nil
}
