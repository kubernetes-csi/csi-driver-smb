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

	fs "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v1beta1"
	fsclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v1beta1"

	smb "github.com/kubernetes-csi/csi-proxy/client/api/smb/v1beta1"
	smbclient "github.com/kubernetes-csi/csi-proxy/client/groups/smb/v1beta1"

	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"
)

var _ CSIProxyMounter = &csiProxyMounter{}

type csiProxyMounterV1Beta struct {
	FsClient  *fsclient.Client
	SMBClient *smbclient.Client
}

func (mounter *csiProxyMounterV1Beta) SMBMount(source, target, fsType string, mountOptions, sensitiveMountOptions []string, volumeID string) error {
	klog.V(4).Infof("SMBMount: remote path: %s. local path: %s", source, target)

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
	normalizedTarget := normalizeWindowsPath(target)
	smbMountRequest := &smb.NewSmbGlobalMappingRequest{
		LocalPath:  normalizedTarget,
		RemotePath: source,
		Username:   mountOptions[0],
		Password:   sensitiveMountOptions[0],
	}
	klog.V(2).Infof("begin to mount %s on %s", source, normalizedTarget)
	if _, err := mounter.SMBClient.NewSmbGlobalMapping(context.Background(), smbMountRequest); err != nil {
		return fmt.Errorf("smb mapping failed with error: %v", err)
	}
	klog.V(2).Infof("mount %s on %s successfully", source, normalizedTarget)
	return nil
}

func (mounter *csiProxyMounterV1Beta) SMBUnmount(target string, volumeID string) error {
	klog.V(4).Infof("SMBUnmount: local path: %s", target)
	// TODO: We need to remove the SMB mapping. The change to remove the
	// directory brings the CSI code in parity with the in-tree.
	return mounter.Rmdir(target)
}

// Mount just creates a soft link at target pointing to source.
func (mounter *csiProxyMounterV1Beta) Mount(source string, target string, fstype string, options []string) error {
	klog.V(4).Infof("Mount: old name: %s. new name: %s", source, target)
	// Mount is called after the format is done.
	// TODO: Confirm that fstype is empty.
	linkRequest := &fs.LinkPathRequest{
		SourcePath: normalizeWindowsPath(source),
		TargetPath: normalizeWindowsPath(target),
	}
	_, err := mounter.FsClient.LinkPath(context.Background(), linkRequest)
	if err != nil {
		return err
	}
	return nil
}

// Rmdir - delete the given directory
// TODO: Call separate rmdir for pod context and plugin context. v1alpha1 for CSI
//
//	proxy does a relaxed check for prefix as c:\var\lib\kubelet, so we can do
//	rmdir with either pod or plugin context.
func (mounter *csiProxyMounterV1Beta) Rmdir(path string) error {
	klog.V(4).Infof("Remove directory: %s", path)
	rmdirRequest := &fs.RmdirRequest{
		Path:    normalizeWindowsPath(path),
		Context: fs.PathContext_POD,
		Force:   true,
	}
	_, err := mounter.FsClient.Rmdir(context.Background(), rmdirRequest)
	if err != nil {
		return err
	}
	return nil
}

// Unmount - Removes the directory - equivalent to unmount on Linux.
func (mounter *csiProxyMounterV1Beta) Unmount(target string) error {
	klog.V(4).Infof("Unmount: %s", target)
	return mounter.Rmdir(target)
}

func (mounter *csiProxyMounterV1Beta) List() ([]mount.MountPoint, error) {
	return []mount.MountPoint{}, fmt.Errorf("List not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) IsMountPointMatch(mp mount.MountPoint, dir string) bool {
	return mp.Path == dir
}

// IsLikelyMountPoint - If the directory does not exists, the function will return os.ErrNotExist error.
//
//	If the path exists, call to CSI proxy will check if its a link, if its a link then existence of target
//	path is checked.
func (mounter *csiProxyMounterV1Beta) IsLikelyNotMountPoint(path string) (bool, error) {
	klog.V(4).Infof("IsLikelyNotMountPoint: %s", path)
	isExists, err := mounter.ExistsPath(path)
	if err != nil {
		return false, err
	}
	if !isExists {
		return true, os.ErrNotExist
	}

	response, err := mounter.FsClient.IsMountPoint(context.Background(),
		&fs.IsMountPointRequest{
			Path: normalizeWindowsPath(path),
		})
	if err != nil {
		return false, err
	}
	return !response.IsMountPoint, nil
}

// IsMountPoint: determines if a directory is a mountpoint.
func (mounter *csiProxyMounterV1Beta) IsMountPoint(file string) (bool, error) {
	isNotMnt, err := mounter.IsLikelyNotMountPoint(file)
	if err != nil {
		return false, err
	}
	return !isNotMnt, nil
}

// CanSafelySkipMountPointCheck always returns false on Windows
func (mounter *csiProxyMounterV1Beta) CanSafelySkipMountPointCheck() bool {
	return false
}

func (mounter *csiProxyMounterV1Beta) PathIsDevice(pathname string) (bool, error) {
	return false, fmt.Errorf("PathIsDevice not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) DeviceOpened(pathname string) (bool, error) {
	return false, fmt.Errorf("DeviceOpened not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) GetDeviceNameFromMount(mountPath, pluginMountDir string) (string, error) {
	return "", fmt.Errorf("GetDeviceNameFromMount not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) MakeRShared(path string) error {
	return fmt.Errorf("MakeRShared not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) MakeFile(pathname string) error {
	return fmt.Errorf("MakeFile not implemented for CSIProxyMounterV1Beta")
}

// MakeDir - Creates a directory. The CSI proxy takes in context information.
// Currently the make dir is only used from the staging code path, hence we call it
// with Plugin context..
func (mounter *csiProxyMounterV1Beta) MakeDir(path string) error {
	klog.V(4).Infof("Make directory: %s", path)
	mkdirReq := &fs.MkdirRequest{
		Path:    normalizeWindowsPath(path),
		Context: fs.PathContext_PLUGIN,
	}
	_, err := mounter.FsClient.Mkdir(context.Background(), mkdirReq)
	if err != nil {
		return err
	}

	return nil
}

// ExistsPath - Checks if a path exists. Unlike util ExistsPath, this call does not perform follow link.
func (mounter *csiProxyMounterV1Beta) ExistsPath(path string) (bool, error) {
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
func (mounter *csiProxyMounterV1Beta) GetAPIVersions() string {
	return fmt.Sprintf(
		"API Versions filesystem: %s, SMB: %s",
		fsclient.Version,
		smbclient.Version,
	)
}

func (mounter *csiProxyMounterV1Beta) EvalHostSymlinks(pathname string) (string, error) {
	return "", fmt.Errorf("EvalHostSymlinks not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) GetMountRefs(pathname string) ([]string, error) {
	return []string{}, fmt.Errorf("GetMountRefs not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) GetFSGroup(pathname string) (int64, error) {
	return -1, fmt.Errorf("GetFSGroup not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) GetSELinuxSupport(pathname string) (bool, error) {
	return false, fmt.Errorf("GetSELinuxSupport not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) GetMode(pathname string) (os.FileMode, error) {
	return 0, fmt.Errorf("GetMode not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) MountSensitive(source string, target string, fstype string, options []string, sensitiveOptions []string) error {
	return fmt.Errorf("MountSensitive not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) MountSensitiveWithoutSystemd(source string, target string, fstype string, options []string, sensitiveOptions []string) error {
	return fmt.Errorf("MountSensitiveWithoutSystemd not implemented for CSIProxyMounterV1Beta")
}

func (mounter *csiProxyMounterV1Beta) MountSensitiveWithoutSystemdWithMountFlags(source string, target string, fstype string, options []string, sensitiveOptions []string, mountFlags []string) error {
	return mounter.MountSensitive(source, target, fstype, options, sensitiveOptions /* sensitiveOptions */)
}

// NewCSIProxyMounter - creates a new CSI Proxy mounter struct which encompassed all the
// clients to the CSI proxy - filesystem, disk and volume clients.
func NewCSIProxyMounterV1Beta() (*csiProxyMounterV1Beta, error) {
	fsClient, err := fsclient.NewClient()
	if err != nil {
		return nil, err
	}
	smbClient, err := smbclient.NewClient()
	if err != nil {
		return nil, err
	}

	return &csiProxyMounterV1Beta{
		FsClient:  fsClient,
		SMBClient: smbClient,
	}, nil
}
