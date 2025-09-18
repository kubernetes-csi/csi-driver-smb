//go:build darwin
// +build darwin

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
	"os"

	mount "k8s.io/mount-utils"
)

func Mount(m *mount.SafeFormatAndMount, source, target, fsType string, options []string, sensitiveMountOptions []string, volumeID string) error {
	return m.MountSensitive(source, target, fsType, options, sensitiveMountOptions)
}

func CleanupSMBMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool, volumeID string) error {
	return mount.CleanupMountPoint(target, m, extensiveMountCheck)
}

func CleanupMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool) error {
	return mount.CleanupMountPoint(target, m, extensiveMountCheck)
}

func preparePublishPath(path string, m *mount.SafeFormatAndMount) error {
	return nil
}

func prepareStagePath(path string, m *mount.SafeFormatAndMount) error {
	return nil
}

func Mkdir(m *mount.SafeFormatAndMount, name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

// HasMountReferences is stubbed for macOS as bind mount inspection is not implemented.
// Always returns false to allow unmounting, but this limits the race condition protection
// available on macOS compared to Linux.
func HasMountReferences(stagingTargetPath string) (bool, error) {
	// macOS implementation could potentially inspect mount points but is not implemented
	// This is a known limitation that reduces race condition protection
	return false, nil
}
