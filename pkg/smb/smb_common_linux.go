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
	"fmt"
	"os"
	"strings"

	mount "k8s.io/mount-utils"
)

// Returns true if the `options` contains password with a special characters, and so "credentials=" needed.
// (see comments for ContainsSpecialCharacter() in pkg/smb/nodeserver.go).
// NB: implementation relies on the format:
// options := []string{fmt.Sprintf("%s=%s", usernameField, username), fmt.Sprintf("%s=%s", passwordField, password)}
func NeedsCredentialsOption(options []string) bool {
	return len(options) == 2 && strings.HasPrefix(options[1], "password=") && ContainsSpecialCharacter(options[1])
}

func Mount(m *mount.SafeFormatAndMount, source, target, fsType string, options, sensitiveMountOptions []string, _ string) error {
	if NeedsCredentialsOption(sensitiveMountOptions) {
		file, err := os.CreateTemp("/tmp/", "*.smb.credentials")
		if err != nil {
			return err
		}
		defer func() {
			file.Close()
			os.Remove(file.Name())
		}()

		for _, option := range sensitiveMountOptions {
			if _, err := file.Write([]byte(fmt.Sprintf("%s\n", option))); err != nil {
				return err
			}
		}

		sensitiveMountOptions = []string{fmt.Sprintf("credentials=%s", file.Name())}
	}
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
