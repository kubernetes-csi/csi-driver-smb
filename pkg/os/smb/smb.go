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

	"github.com/kubernetes-csi/csi-driver-smb/pkg/util"
	"k8s.io/klog/v2"
)

// SMBGlobalMappingStatus describes the observed status of a Windows SMB
// global mapping for a remote UNC path.
type SMBGlobalMappingStatus string

const (
	// SMBGlobalMappingStatusNotFound means no mapping exists for the remote path.
	SMBGlobalMappingStatusNotFound SMBGlobalMappingStatus = "NotFound"
	// SMBGlobalMappingStatusOK means the mapping exists and reports a healthy state.
	SMBGlobalMappingStatusOK SMBGlobalMappingStatus = "OK"
	// SMBGlobalMappingStatusDisconnected means the mapping object exists but is stale/unhealthy.
	SMBGlobalMappingStatusDisconnected SMBGlobalMappingStatus = "Disconnected"
	// SMBGlobalMappingStatusOther covers any other existing non-OK state.
	SMBGlobalMappingStatusOther SMBGlobalMappingStatus = "Other"
)

func parseSMBGlobalMappingStatus(out string) SMBGlobalMappingStatus {
	switch strings.ToLower(strings.TrimSpace(out)) {
	case "":
		return SMBGlobalMappingStatusNotFound
	case strings.ToLower(string(SMBGlobalMappingStatusOK)):
		return SMBGlobalMappingStatusOK
	case strings.ToLower(string(SMBGlobalMappingStatusDisconnected)):
		return SMBGlobalMappingStatusDisconnected
	case strings.ToLower(string(SMBGlobalMappingStatusNotFound)):
		return SMBGlobalMappingStatusNotFound
	default:
		return SMBGlobalMappingStatusOther
	}
}

// GetSmbGlobalMappingStatus returns the current SMB global mapping status for
// the given remote UNC path.
func GetSmbGlobalMappingStatus(remotePath string) (SMBGlobalMappingStatus, error) {
	cmdLine := `$mapping = Get-SmbGlobalMapping -ErrorAction Stop | Where-Object { $_.RemotePath -eq $Env:smbremotepath } | Select-Object -First 1; if ($null -eq $mapping) { 'NotFound' } else { $mapping.Status }`
	cmdEnv := fmt.Sprintf("smbremotepath=%s", remotePath)
	out, err := util.RunPowershellCmd(cmdLine, cmdEnv)
	if err != nil {
		return SMBGlobalMappingStatusNotFound, fmt.Errorf("error checking smb mapping. cmd %s, output: %s, err: %v", remotePath, string(out), err)
	}
	return parseSMBGlobalMappingStatus(string(out)), nil
}

func IsSmbMapped(remotePath string) (bool, error) {
	status, err := GetSmbGlobalMappingStatus(remotePath)
	if err != nil {
		return false, err
	}
	return status == SMBGlobalMappingStatusOK, nil
}

func newSmbGlobalMappingCmd(requirePrivacy bool) string {
	requirePrivacyValue := "$true"
	if !requirePrivacy {
		requirePrivacyValue = "$false"
	}
	return fmt.Sprintf(`$PWord = ConvertTo-SecureString -String $Env:smbpassword -AsPlainText -Force`+
		`;$Credential = New-Object -TypeName System.Management.Automation.PSCredential -ArgumentList $Env:smbuser, $PWord`+
		`;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Credential $Credential -RequirePrivacy %s`, requirePrivacyValue)
}

func NewSmbGlobalMapping(remotePath, username, password string, requirePrivacy bool) error {
	// use PowerShell Environment Variables to store user input string to prevent command line injection
	// https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_environment_variables?view=powershell-5.1
	cmdLine := newSmbGlobalMappingCmd(requirePrivacy)

	klog.V(2).Infof("begin to run NewSmbGlobalMapping with %s, %s, requirePrivacy=%v", remotePath, username, requirePrivacy)
	if output, err := util.RunPowershellCmd(cmdLine, fmt.Sprintf("smbuser=%s", username),
		fmt.Sprintf("smbpassword=%s", password),
		fmt.Sprintf("smbremotepath=%s", remotePath)); err != nil {
		return fmt.Errorf("NewSmbGlobalMapping failed. output: %q, err: %v", string(output), err)
	}
	return nil
}

func RemoveSmbGlobalMapping(remotePath string) error {
	remotePath = strings.TrimSuffix(remotePath, `\`)
	cmd := `Remove-SmbGlobalMapping -RemotePath $Env:smbremotepath -Force`
	klog.V(2).Infof("begin to run RemoveSmbGlobalMapping with %s", remotePath)
	if output, err := util.RunPowershellCmd(cmd, fmt.Sprintf("smbremotepath=%s", remotePath)); err != nil {
		return fmt.Errorf("UnmountSmbShare failed. output: %q, err: %v", string(output), err)
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
						if remoteServerPath == remoteServer {
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
