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

package mounter

import (
	"context"
	"fmt"
	"net"
	"os"
	filepath "path/filepath"
	"strings"

	fs "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v1"
	fsclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v1"

	smb "github.com/kubernetes-csi/csi-proxy/client/api/smb/v1"
	smbclient "github.com/kubernetes-csi/csi-proxy/client/groups/smb/v1"

	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

// CSIProxyMounter extends the mount.Interface interface with CSI Proxy methods.
type CSIProxyMounter interface {
	mount.Interface

	SMBMount(source, target, fsType string, mountOptions, sensitiveMountOptions []string, volumeID string) error
	SMBUnmount(target string, volumeID string) error
	MakeDir(path string) error
	Rmdir(path string) error
	IsMountPointMatch(mp mount.MountPoint, dir string) bool
	ExistsPath(path string) (bool, error)
	GetAPIVersions() string
	EvalHostSymlinks(pathname string) (string, error)
}

var _ CSIProxyMounter = &csiProxyMounter{}

type csiProxyMounter struct {
	FsClient                      *fsclient.Client
	SMBClient                     *smbclient.Client
	RemoveSMBMappingDuringUnmount bool
}

func normalizeWindowsPath(path string) string {
	normalizedPath := strings.Replace(path, "/", "\\", -1)
	if strings.HasPrefix(normalizedPath, "\\") {
		normalizedPath = "c:" + normalizedPath
	}
	return normalizedPath
}

func (mounter *csiProxyMounter) SMBMount(source, target, fsType string, mountOptions, sensitiveMountOptions []string, volumeID string) error {
	klog.V(2).Infof("SMBMount: remote path: %s local path: %s", source, target)

	if len(mountOptions) == 0 || len(sensitiveMountOptions) == 0 {
		return fmt.Errorf("empty mountOptions(len: %d) or sensitiveMountOptions(len: %d) is not allowed", len(mountOptions), len(sensitiveMountOptions))
	}

	parentDir := filepath.Dir(target)
	parentExists, err := mounter.ExistsPath(parentDir)
	if err != nil {
		return fmt.Errorf("parent dir: %s exist check failed with err: %v", parentDir, err)
	}

	if !parentExists {
		klog.V(2).Infof("Parent directory %s does not exists. Creating the directory", parentDir)
		if err := mounter.MakeDir(parentDir); err != nil {
			return fmt.Errorf("create of parent dir: %s dailed with error: %v", parentDir, err)
		}
	}

	parts := strings.FieldsFunc(source, Split)
	if len(parts) > 0 && strings.HasSuffix(parts[0], "svc.cluster.local") {
		domainName := parts[0]
		klog.V(2).Infof("begin to replace hostname(%s) with IP for source(%s)", domainName, source)
		ip, err := net.ResolveIPAddr("ip4", domainName)
		if err != nil {
			klog.Warningf("could not resolve name to IPv4 address for host %s, failed with error: %v", domainName, err)
		} else {
			klog.V(2).Infof("resolve the name of host %s to IPv4 address: %s", domainName, ip.String())
			source = strings.Replace(source, domainName, ip.String(), 1)
		}
	}

	source = strings.Replace(source, "/", "\\", -1)
	source = strings.TrimSuffix(source, "\\")
	mappingPath, err := getRootMappingPath(source)
	if mounter.RemoveSMBMappingDuringUnmount && err != nil {
		return fmt.Errorf("getRootMappingPath(%s) failed with error: %v", source, err)
	}
	unlock := lock(mappingPath)
	defer unlock()

	normalizedTarget := normalizeWindowsPath(target)
	smbMountRequest := &smb.NewSmbGlobalMappingRequest{
		LocalPath:  normalizedTarget,
		RemotePath: source,
		Username:   mountOptions[0],
		Password:   sensitiveMountOptions[0],
	}
	klog.V(2).Infof("begin to NewSmbGlobalMapping %s on %s", source, normalizedTarget)
	if _, err := mounter.SMBClient.NewSmbGlobalMapping(context.Background(), smbMountRequest); err != nil {
		return fmt.Errorf("NewSmbGlobalMapping(%s, %s) failed with error: %v", source, normalizedTarget, err)
	}
	klog.V(2).Infof("NewSmbGlobalMapping %s on %s successfully", source, normalizedTarget)

	if mounter.RemoveSMBMappingDuringUnmount {
		if err := incementVolumeIDReferencesCount(mappingPath, source, volumeID); err != nil {
			return fmt.Errorf("incementRemotePathReferencesCount(%s, %s, %s) failed with error: %v", mappingPath, source, volumeID, err)
		}
	}
	return nil
}

func (mounter *csiProxyMounter) SMBUnmount(target, volumeID string) error {
	klog.V(4).Infof("SMBUnmount: local path: %s", target)

	if remotePath, err := os.Readlink(target); err != nil {
		klog.Warningf("SMBUnmount: can't get remote path: %v", err)
	} else {
		remotePath = strings.TrimSuffix(remotePath, "\\")
		mappingPath, err := getRootMappingPath(remotePath)
		if mounter.RemoveSMBMappingDuringUnmount && err != nil {
			return fmt.Errorf("getRootMappingPath(%s) failed with error: %v", remotePath, err)
		}
		klog.V(4).Infof("SMBUnmount: remote path: %s, mapping path: %s", remotePath, mappingPath)

		if mounter.RemoveSMBMappingDuringUnmount {
			unlock := lock(mappingPath)
			defer unlock()

			if err := decrementVolumeIDReferencesCount(mappingPath, volumeID); err != nil {
				return fmt.Errorf("decrementRemotePathReferencesCount(%s, %s) failed with error: %v", mappingPath, volumeID, err)
			}
			count := getVolumeIDReferencesCount(mappingPath)
			if count == 0 {
				smbUnmountRequest := &smb.RemoveSmbGlobalMappingRequest{
					RemotePath: remotePath,
				}
				klog.V(2).Infof("begin to RemoveSmbGlobalMapping %s on %s", remotePath, target)
				if _, err := mounter.SMBClient.RemoveSmbGlobalMapping(context.Background(), smbUnmountRequest); err != nil {
					return fmt.Errorf("RemoveSmbGlobalMapping failed with error: %v", err)
				}
				klog.V(2).Infof("RemoveSmbGlobalMapping %s on %s successfully", remotePath, target)
			} else {
				klog.Infof("SMBUnmount: found %d links to %s", count, mappingPath)
			}
		}
	}

	return mounter.Rmdir(target)
}

// Mount just creates a soft link at target pointing to source.
func (mounter *csiProxyMounter) Mount(source string, target string, fstype string, options []string) error {
	klog.V(4).Infof("Mount: old name: %s. new name: %s", source, target)
	// Mount is called after the format is done.
	// TODO: Confirm that fstype is empty.
	linkRequest := &fs.CreateSymlinkRequest{
		SourcePath: normalizeWindowsPath(source),
		TargetPath: normalizeWindowsPath(target),
	}
	_, err := mounter.FsClient.CreateSymlink(context.Background(), linkRequest)
	if err != nil {
		return err
	}
	return nil
}

func Split(r rune) bool {
	return r == ' ' || r == '/'
}

// Rmdir - delete the given directory
// TODO: Call separate rmdir for pod context and plugin context. v1alpha1 for CSI
//
//	proxy does a relaxed check for prefix as c:\var\lib\kubelet, so we can do
//	rmdir with either pod or plugin context.
func (mounter *csiProxyMounter) Rmdir(path string) error {
	klog.V(4).Infof("Remove directory: %s", path)
	rmdirRequest := &fs.RmdirRequest{
		Path:  normalizeWindowsPath(path),
		Force: true,
	}
	_, err := mounter.FsClient.Rmdir(context.Background(), rmdirRequest)
	if err != nil {
		return err
	}
	return nil
}

// Unmount - Removes the directory - equivalent to unmount on Linux.
func (mounter *csiProxyMounter) Unmount(target string) error {
	klog.V(4).Infof("Unmount: %s", target)
	return mounter.Rmdir(target)
}

func (mounter *csiProxyMounter) List() ([]mount.MountPoint, error) {
	return []mount.MountPoint{}, fmt.Errorf("List not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) IsMountPointMatch(mp mount.MountPoint, dir string) bool {
	return mp.Path == dir
}

// IsLikelyMountPoint - If the directory does not exists, the function will return os.ErrNotExist error.
//
//	If the path exists, call to CSI proxy will check if its a link, if its a link then existence of target
//	path is checked.
func (mounter *csiProxyMounter) IsLikelyNotMountPoint(path string) (bool, error) {
	klog.V(4).Infof("IsLikelyNotMountPoint: %s", path)
	isExists, err := mounter.ExistsPath(path)
	if err != nil {
		return false, err
	}
	if !isExists {
		return true, os.ErrNotExist
	}

	response, err := mounter.FsClient.IsSymlink(context.Background(),
		&fs.IsSymlinkRequest{
			Path: normalizeWindowsPath(path),
		})
	if err != nil {
		return false, err
	}
	return !response.IsSymlink, nil
}

// IsMountPoint: determines if a directory is a mountpoint.
func (mounter *csiProxyMounter) IsMountPoint(file string) (bool, error) {
	isNotMnt, err := mounter.IsLikelyNotMountPoint(file)
	if err != nil {
		return false, err
	}
	return !isNotMnt, nil
}

// CanSafelySkipMountPointCheck always returns false on Windows
func (mounter *csiProxyMounter) CanSafelySkipMountPointCheck() bool {
	return false
}

func (mounter *csiProxyMounter) PathIsDevice(pathname string) (bool, error) {
	return false, fmt.Errorf("PathIsDevice not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) DeviceOpened(pathname string) (bool, error) {
	return false, fmt.Errorf("DeviceOpened not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) GetDeviceNameFromMount(mountPath, pluginMountDir string) (string, error) {
	return "", fmt.Errorf("GetDeviceNameFromMount not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) MakeRShared(path string) error {
	return fmt.Errorf("MakeRShared not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) MakeFile(pathname string) error {
	return fmt.Errorf("MakeFile not implemented for CSIProxyMounter")
}

// MakeDir - Creates a directory. The CSI proxy takes in context information.
// Currently the make dir is only used from the staging code path, hence we call it
// with Plugin context..
func (mounter *csiProxyMounter) MakeDir(path string) error {
	klog.V(4).Infof("Make directory: %s", path)
	mkdirReq := &fs.MkdirRequest{
		Path: normalizeWindowsPath(path),
	}
	_, err := mounter.FsClient.Mkdir(context.Background(), mkdirReq)
	if err != nil {
		return err
	}

	return nil
}

// ExistsPath - Checks if a path exists. Unlike util ExistsPath, this call does not perform follow link.
func (mounter *csiProxyMounter) ExistsPath(path string) (bool, error) {
	klog.V(4).Infof("Exists path: %s", path)
	isExistsResponse, err := mounter.FsClient.PathExists(context.Background(),
		&fs.PathExistsRequest{
			Path: normalizeWindowsPath(path),
		})
	if err != nil {
		return false, err
	}
	return isExistsResponse.Exists, err
}

// GetAPIVersions returns the versions of the client APIs this mounter is using.
func (mounter *csiProxyMounter) GetAPIVersions() string {
	return fmt.Sprintf(
		"API Versions filesystem: %s, SMB: %s",
		fsclient.Version,
		smbclient.Version,
	)
}

func (mounter *csiProxyMounter) EvalHostSymlinks(pathname string) (string, error) {
	return "", fmt.Errorf("EvalHostSymlinks not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) GetMountRefs(pathname string) ([]string, error) {
	return []string{}, fmt.Errorf("GetMountRefs not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) GetFSGroup(pathname string) (int64, error) {
	return -1, fmt.Errorf("GetFSGroup not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) GetSELinuxSupport(pathname string) (bool, error) {
	return false, fmt.Errorf("GetSELinuxSupport not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) GetMode(pathname string) (os.FileMode, error) {
	return 0, fmt.Errorf("GetMode not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) MountSensitive(source string, target string, fstype string, options []string, sensitiveOptions []string) error {
	return fmt.Errorf("MountSensitive not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) MountSensitiveWithoutSystemd(source string, target string, fstype string, options []string, sensitiveOptions []string) error {
	return fmt.Errorf("MountSensitiveWithoutSystemd not implemented for CSIProxyMounter")
}

func (mounter *csiProxyMounter) MountSensitiveWithoutSystemdWithMountFlags(source string, target string, fstype string, options []string, sensitiveOptions []string, mountFlags []string) error {
	return mounter.MountSensitive(source, target, fstype, options, sensitiveOptions /* sensitiveOptions */)
}

// NewCSIProxyMounter - creates a new CSI Proxy mounter struct which encompassed all the
// clients to the CSI proxy - filesystem, disk and volume clients.
func NewCSIProxyMounter(removeSMBMappingDuringUnmount bool) (*csiProxyMounter, error) {
	fsClient, err := fsclient.NewClient()
	if err != nil {
		return nil, err
	}
	smbClient, err := smbclient.NewClient()
	if err != nil {
		return nil, err
	}

	return &csiProxyMounter{
		FsClient:                      fsClient,
		SMBClient:                     smbClient,
		RemoveSMBMappingDuringUnmount: removeSMBMappingDuringUnmount,
	}, nil
}

func NewSafeMounter(enableWindowsHostProcess, removeSMBMappingDuringUnmount bool) (*mount.SafeFormatAndMount, error) {
	if enableWindowsHostProcess {
		klog.V(2).Infof("using windows host process mounter")
		return &mount.SafeFormatAndMount{
			Interface: NewWinMounter(),
			Exec:      utilexec.New(),
		}, nil
	}
	csiProxyMounter, err := NewCSIProxyMounter(removeSMBMappingDuringUnmount)
	if err == nil {
		klog.V(2).Infof("using CSIProxyMounterV1, %s", csiProxyMounter.GetAPIVersions())
		return &mount.SafeFormatAndMount{
			Interface: csiProxyMounter,
			Exec:      utilexec.New(),
		}, nil
	}

	klog.V(2).Infof("failed to connect to csi-proxy v1 with error: %v, will try with v1Beta", err)
	csiProxyMounterV1Beta, err := NewCSIProxyMounterV1Beta()
	if err == nil {
		klog.V(2).Infof("using CSIProxyMounterV1beta, %s", csiProxyMounterV1Beta.GetAPIVersions())
		return &mount.SafeFormatAndMount{
			Interface: csiProxyMounterV1Beta,
			Exec:      utilexec.New(),
		}, nil
	}

	klog.Errorf("failed to connect to csi-proxy v1beta with error: %v", err)
	return nil, err
}
