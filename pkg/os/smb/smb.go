//go:build windows
// +build windows

/*
Copyright 2023 The Kubernetes Authors.

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
	"path/filepath"
	"strings"

	"github.com/kubernetes-csi/csi-proxy/v2/pkg/wmi"
	"k8s.io/klog/v2"
)

// CanonicalizeSMBRemotePath normalizes an SMB remote UNC path into a stable
// canonical representation by converting separators, removing all trailing
// backslashes, and folding case.
func CanonicalizeSMBRemotePath(remotePath string) string {
	remotePath = strings.ReplaceAll(remotePath, "/", "\\")
	remotePath = strings.TrimRight(remotePath, `\`)
	return strings.ToLower(remotePath)
}

// SMBGlobalMappingStatus describes the observed status of a Windows SMB
// global mapping for a remote UNC path.
type SMBGlobalMappingStatus string

const (
	SMBGlobalMappingStatusNotFound     SMBGlobalMappingStatus = "NotFound"
	SMBGlobalMappingStatusOK           SMBGlobalMappingStatus = "OK"
	SMBGlobalMappingStatusDisconnected SMBGlobalMappingStatus = "Disconnected"
	SMBGlobalMappingStatusOther        SMBGlobalMappingStatus = "Other"
)

func smbGlobalMappingStatusFromWMI(status uint32) SMBGlobalMappingStatus {
	switch status {
	case wmi.SmbMappingStatusOK:
		return SMBGlobalMappingStatusOK
	case wmi.SmbMappingStatusDisconnected:
		return SMBGlobalMappingStatusDisconnected
	default:
		return SMBGlobalMappingStatusOther
	}
}

// newSmbGlobalMappingWMI is the low-level WMI create call; overridden in unit tests.
var newSmbGlobalMappingWMI = wmi.NewSmbGlobalMapping
var querySmbGlobalMappingWMI = wmi.QuerySmbGlobalMappingByRemotePath
var getSmbGlobalMappingStatusWMI = wmi.GetSmbGlobalMappingStatus

func GetSmbGlobalMappingStatus(remotePath string) (SMBGlobalMappingStatus, error) {
	remotePath = CanonicalizeSMBRemotePath(remotePath)
	status := SMBGlobalMappingStatusNotFound
	err := wmi.WithCOMThread(func() error {
		return wmi.WithScope(func(scope *wmi.Scope) error {
			inst, err := querySmbGlobalMappingWMI(scope, remotePath)
			if err != nil {
				return err
			}
			mappingStatus, err := getSmbGlobalMappingStatusWMI(inst)
			if err != nil {
				return err
			}
			status = smbGlobalMappingStatusFromWMI(mappingStatus)
			return nil
		})
	})
	return status, wmi.IgnoreNotFound(err)
}

func IsSmbMapped(remotePath string) (bool, error) {
	status, err := GetSmbGlobalMappingStatus(remotePath)
	return status == SMBGlobalMappingStatusOK, err
}

func NewSmbGlobalMapping(remotePath, username, password string, requirePrivacy bool) error {
	remotePath = CanonicalizeSMBRemotePath(remotePath)
	klog.V(2).Infof("begin to run NewSmbGlobalMapping with %s, %s, requirePrivacy=%v", remotePath, username, requirePrivacy)
	err := wmi.WithCOMThread(func() error {
		return newSmbGlobalMappingWMI(remotePath, username, password, requirePrivacy)
	})
	if err != nil {
		return fmt.Errorf("NewSmbGlobalMapping failed. err: %v", err)
	}
	return nil
}

func RemoveSmbGlobalMapping(remotePath string) error {
	remotePath = CanonicalizeSMBRemotePath(remotePath)
	klog.V(2).Infof("begin to run RemoveSmbGlobalMapping with %s", remotePath)
	err := wmi.WithCOMThread(func() error {
		return wmi.WithScope(func(scope *wmi.Scope) error {
			return wmi.RemoveSmbGlobalMappingByRemotePath(scope, remotePath)
		})
	})
	if err != nil {
		return fmt.Errorf("UnmountSmbShare failed. err: %v", err)
	}
	return nil
}

// GetRemoteServerFromTarget- gets the remote server path given a mount point, the function is recursive until it find the remote server or errors out
func GetRemoteServerFromTarget(mount string) (string, error) {
	target, err := os.Readlink(mount)
	klog.V(2).Infof("read link for mount %s, target: %s", mount, target)
	if err != nil || len(target) == 0 {
		return "", fmt.Errorf("error reading link for mount %s. target %s err: %v", mount, target, err)
	}
	return strings.TrimSpace(target), nil
}

// CheckForDuplicateSMBMounts checks if there is any other SMB mount exists on the same remote server
func CheckForDuplicateSMBMounts(dir, mount, remoteServer string) (bool, error) {
	remoteServer = CanonicalizeSMBRemotePath(remoteServer)
	files, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, file := range files {
		klog.V(6).Infof("checking file %s", file.Name())
		if file.IsDir() {
			globalMountPath := filepath.Join(dir, file.Name(), "globalmount")
			if strings.EqualFold(filepath.Clean(globalMountPath), filepath.Clean(mount)) {
				klog.V(2).Infof("skip current mount path %s", mount)
			} else {
				fileInfo, err := os.Lstat(globalMountPath)
				// check if the file is a symlink, if yes, check if it is pointing to the same remote server
				if err == nil && fileInfo.Mode()&os.ModeSymlink != 0 {
					remoteServerPath, err := GetRemoteServerFromTarget(globalMountPath)
					klog.V(2).Infof("checking remote server path %s on local path %s", remoteServerPath, globalMountPath)
					if err == nil {
						if CanonicalizeSMBRemotePath(remoteServerPath) == remoteServer {
							return true, nil
						}
					} else {
						klog.Errorf("GetRemoteServerFromTarget(%s) failed with %v", globalMountPath, err)
					}
				}
			}
		}
	}
	return false, err
}
