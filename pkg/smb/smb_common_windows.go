//go:build windows
// +build windows

/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"os"

	"github.com/kubernetes-csi/csi-driver-smb/pkg/mounter"
	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"
)

func Mount(m *mount.SafeFormatAndMount, source, target, fsType string, mountOptions, sensitiveMountOptions []string, volumeID string) error {
	if proxy, ok := m.Interface.(mounter.CSIProxyMounter); ok {
		return proxy.SMBMount(source, target, fsType, mountOptions, sensitiveMountOptions, volumeID)
	}
	return fmt.Errorf("could not cast to csi proxy class")
}

// CleanupSMBMountPoint - In windows CSI proxy call to umount is used to unmount the SMB.
// The clean up mount point point calls is supposed for fix the corrupted directories as well.
// For alpha CSI proxy integration, we only do an unmount.
func CleanupSMBMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool, volumeID string) error {
	if proxy, ok := m.Interface.(mounter.CSIProxyMounter); ok {
		return proxy.SMBUnmount(target, volumeID)
	}
	return fmt.Errorf("could not cast to csi proxy class")
}

func CleanupMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool) error {
	if proxy, ok := m.Interface.(mounter.CSIProxyMounter); ok {
		return proxy.Rmdir(target)
	}
	return fmt.Errorf("could not cast to csi proxy class")
}

func removeDir(path string, m *mount.SafeFormatAndMount) error {
	if proxy, ok := m.Interface.(mounter.CSIProxyMounter); ok {
		isExists, err := proxy.ExistsPath(path)
		if err != nil {
			return err
		}

		if isExists {
			klog.V(4).Infof("Removing path: %s", path)
			if err = proxy.Rmdir(path); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("could not cast to csi proxy class")
}

// preparePublishPath - In case of windows, the publish code path creates a soft link
// from global stage path to the publish path. But kubelet creates the directory in advance.
// We work around this issue by deleting the publish path then recreating the link.
func preparePublishPath(path string, m *mount.SafeFormatAndMount) error {
	return removeDir(path, m)
}

func prepareStagePath(path string, m *mount.SafeFormatAndMount) error {
	return removeDir(path, m)
}

func Mkdir(m *mount.SafeFormatAndMount, name string, perm os.FileMode) error {
	if proxy, ok := m.Interface.(mounter.CSIProxyMounter); ok {
		return proxy.MakeDir(name)
	}
	return fmt.Errorf("could not cast to csi proxy class")
}
