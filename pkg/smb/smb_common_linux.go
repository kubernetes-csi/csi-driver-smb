//go:build linux
// +build linux

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
	"bufio"
	"fmt"
	"os"
	"strings"

	mount "k8s.io/mount-utils"
)

func Mount(m *mount.SafeFormatAndMount, source, target, fsType string, options, sensitiveMountOptions []string, _ string) error {
	return m.MountSensitive(source, target, fsType, options, sensitiveMountOptions)
}

func CleanupSMBMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool, _ string) error {
	return mount.CleanupMountPoint(target, m, extensiveMountCheck)
}

func CleanupMountPoint(m *mount.SafeFormatAndMount, target string, extensiveMountCheck bool) error {
	return mount.CleanupMountPoint(target, m, extensiveMountCheck)
}

func preparePublishPath(_ string, _ *mount.SafeFormatAndMount) error {
	return nil
}

func prepareStagePath(_ string, _ *mount.SafeFormatAndMount) error {
	return nil
}

func Mkdir(_ *mount.SafeFormatAndMount, name string, perm os.FileMode) error {
	return os.Mkdir(name, perm)
}

func HasMountReferences(stagingTargetPath string) (bool, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to open /proc/mounts: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			mountPoint := fields[1]
			if strings.HasPrefix(mountPoint, stagingTargetPath) && mountPoint != stagingTargetPath {
				return true, nil
			}
		}
	}
	return false, nil
}
