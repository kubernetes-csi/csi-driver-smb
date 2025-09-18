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
	"path/filepath"
	"strings"
	"time"

	"k8s.io/klog/v2"
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

// HasMountReferences checks if the staging path has any bind mount references.
// Uses atomic double-check pattern to prevent race conditions during unstaging.
func HasMountReferences(stagingTargetPath string) (bool, error) {
	const maxRetries = 3
	const baseDelay = 50 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff to allow concurrent operations to settle
			delay := baseDelay * time.Duration(1<<(attempt-1))
			klog.V(4).Infof("HasMountReferences: retry %d after %v for path %s", attempt, delay, stagingTargetPath)
			time.Sleep(delay)
		}

		// First check: scan /proc/mounts for references
		hasRefs, err := checkMountReferencesOnce(stagingTargetPath)
		if err != nil {
			if attempt == maxRetries-1 {
				return false, fmt.Errorf("failed to check mount references after %d attempts: %v", maxRetries, err)
			}
			klog.V(4).Infof("HasMountReferences: attempt %d failed, retrying: %v", attempt, err)
			continue
		}

		if !hasRefs {
			// Double-check: verify no references appeared during our check
			doubleCheck, err := checkMountReferencesOnce(stagingTargetPath)
			if err != nil {
				if attempt == maxRetries-1 {
					return false, fmt.Errorf("failed double-check mount references: %v", err)
				}
				continue
			}

			if !doubleCheck {
				// Consistent result: no references found
				klog.V(4).Infof("HasMountReferences: confirmed no references for %s", stagingTargetPath)
				return false, nil
			}
			// Double-check found references, retry
			klog.V(4).Infof("HasMountReferences: double-check detected new references for %s", stagingTargetPath)
		}

		// References found or inconsistent state, but let's verify it's stable
		if hasRefs {
			klog.V(4).Infof("HasMountReferences: found references for %s", stagingTargetPath)
			return true, nil
		}
	}

	// After all retries, assume references exist to be safe
	klog.V(2).Infof("HasMountReferences: assuming references exist for %s after %d retries (fail-safe)", stagingTargetPath, maxRetries)
	return true, nil
}

// checkMountReferencesOnce performs a single atomic check of /proc/mounts
func checkMountReferencesOnce(stagingTargetPath string) (bool, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to open /proc/mounts: %v", err)
	}
	defer f.Close()

	// Normalize the staging path for comparison
	cleanStagingPath, err := filepath.Abs(stagingTargetPath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for %s: %v", stagingTargetPath, err)
	}
	cleanStagingPath = filepath.Clean(cleanStagingPath)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 6 {
			mountSource := fields[0]
			mountPoint := fields[1]

			// Check if this is a potential bind mount reference
			if isBindMountReference(cleanStagingPath, mountPoint, mountSource) {
				klog.V(4).Infof("checkMountReferencesOnce: found reference %s -> %s (source: %s)",
					cleanStagingPath, mountPoint, mountSource)
				return true, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading /proc/mounts: %v", err)
	}

	return false, nil
}

// isBindMountReference determines if a mount point is a bind mount reference to the staging path.
// It uses multiple validation techniques to avoid false positives from simple string matching.
func isBindMountReference(stagingPath, mountPoint, mountSource string) bool {
	// Clean and normalize both paths for accurate comparison
	cleanMountPoint, err := filepath.Abs(mountPoint)
	if err != nil {
		// If we can't clean the mount point, skip it to be safe
		klog.V(4).Infof("isBindMountReference: failed to clean mount point %s: %v", mountPoint, err)
		return false
	}
	cleanMountPoint = filepath.Clean(cleanMountPoint)

	// Skip if it's the same path (not a reference, it's the staging mount itself)
	if cleanMountPoint == stagingPath {
		return false
	}

	// Method 1: Check if mount point is a proper subdirectory of staging path
	if isProperSubdirectory(stagingPath, cleanMountPoint) {
		klog.V(4).Infof("isBindMountReference: %s is subdirectory of %s", cleanMountPoint, stagingPath)
		return true
	}

	// Method 2: Check if mount source indicates a bind mount from staging path
	// For bind mounts, the source often matches the staging path or subdirectory
	if strings.HasPrefix(mountSource, stagingPath) {
		// Validate this is a proper path hierarchy relationship
		if isProperSubdirectory(stagingPath, mountSource) || mountSource == stagingPath {
			klog.V(4).Infof("isBindMountReference: mount source %s originates from staging path %s",
				mountSource, stagingPath)
			return true
		}
	}

	// Method 3: Additional check for bind mounts where source and target match staging hierarchy
	// This catches cases where both source and target are related to our staging path
	cleanMountSource, err := filepath.Abs(mountSource)
	if err == nil {
		cleanMountSource = filepath.Clean(cleanMountSource)
		if (cleanMountSource == stagingPath || isProperSubdirectory(stagingPath, cleanMountSource)) &&
			(cleanMountPoint != stagingPath && isProperSubdirectory(stagingPath, cleanMountPoint)) {
			klog.V(4).Infof("isBindMountReference: bind mount detected - source %s and target %s both relate to staging path %s",
				cleanMountSource, cleanMountPoint, stagingPath)
			return true
		}
	}

	return false
}

// isProperSubdirectory checks if child is a proper subdirectory of parent.
// It uses path hierarchy validation to avoid false positives from string prefix matching.
func isProperSubdirectory(parent, child string) bool {
	// Ensure both paths are clean and absolute
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	// Child must be longer than parent to be a subdirectory
	if len(child) <= len(parent) {
		return false
	}

	// Check if child starts with parent
	if !strings.HasPrefix(child, parent) {
		return false
	}

	// Validate that the relationship is at a path boundary
	// This prevents false positives like "/path/vol1" matching "/path/vol10"
	remainder := child[len(parent):]

	// The remainder must start with a path separator to be a valid subdirectory
	if !strings.HasPrefix(remainder, string(filepath.Separator)) {
		return false
	}

	return true
}
