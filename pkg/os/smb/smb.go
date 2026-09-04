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
	"strconv"
	"strings"

	"github.com/kubernetes-csi/csi-driver-smb/pkg/util"
	"k8s.io/klog/v2"
)

func IsSmbMapped(remotePath string) (bool, error) {
	cmdLine := `$(Get-SmbGlobalMapping -RemotePath $Env:smbremotepath -ErrorAction Stop).Status`
	cmdEnv := fmt.Sprintf("smbremotepath=%s", remotePath)
	out, err := util.RunPowershellCmd(cmdLine, cmdEnv)
	if err != nil {
		return false, fmt.Errorf("error checking smb mapping. cmd %s, output: %s, err: %v", remotePath, string(out), err)
	}

	if len(out) == 0 || !strings.EqualFold(strings.TrimSpace(string(out)), "OK") {
		return false, nil
	}
	return true, nil
}

var supportedGlobalMappingAdditionalParams = []string{
	"Persistent",
	"RequireIntegrity",
	"RequirePrivacy",
	"UseWriteThrough",
	"FullAccess",
	"DenyAccess",
	"TransportType",
	"SkipCertificateCheck",
	"CompressNetworkTraffic",
	"BlockNTLM",
	"TcpPort",
	"QuicPort",
	"RdmaPort",
}

func splitGlobalMappingListValue(value string) ([]string, error) {
	parts := strings.Split(value, ";")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("list values must not contain empty entries")
		}
		result = append(result, part)
	}
	return result, nil
}

func parseGlobalMappingAdditionalParams(requirePrivacy bool, globalMappingAdditionalParams string) ([]string, error) {
	globalMappingAdditionalParams = strings.TrimSpace(globalMappingAdditionalParams)
	if globalMappingAdditionalParams == "" {
		return []string{fmt.Sprintf("smbopt_requireprivacy=%t", requirePrivacy)}, nil
	}

	seen := map[string]struct{}{}
	envs := []string{}
	for _, rawPart := range strings.Split(globalMappingAdditionalParams, ",") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			return nil, fmt.Errorf("global mapping additional params contain an empty entry")
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid global mapping additional param %q, expected key=value", part)
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid global mapping additional param %q, key and value must be non-empty", part)
		}
		normalizedKey := strings.ToLower(key)
		if _, ok := seen[normalizedKey]; ok {
			return nil, fmt.Errorf("duplicate global mapping additional param %q", key)
		}

		switch normalizedKey {
		case "persistent", "requireintegrity", "requireprivacy", "usewritethrough", "skipcertificatecheck", "compressnetworktraffic", "blockntlm":
			parsedValue, err := strconv.ParseBool(strings.ToLower(value))
			if err != nil {
				return nil, fmt.Errorf("invalid boolean value %q for %s", value, key)
			}
			envs = append(envs, fmt.Sprintf("smbopt_%s=%t", normalizedKey, parsedValue))
		case "fullaccess", "denyaccess":
			items, err := splitGlobalMappingListValue(value)
			if err != nil {
				return nil, fmt.Errorf("invalid list value %q for %s: %v", value, key, err)
			}
			envs = append(envs, fmt.Sprintf("smbopt_%s=%s", normalizedKey, strings.Join(items, ";")))
		case "transporttype":
			envs = append(envs, fmt.Sprintf("smbopt_%s=%s", normalizedKey, value))
		case "tcpport", "quicport", "rdmaport":
			parsedValue, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid uint16 value %q for %s", value, key)
			}
			envs = append(envs, fmt.Sprintf("smbopt_%s=%d", normalizedKey, parsedValue))
		default:
			return nil, fmt.Errorf("unsupported global mapping additional param %q, supported params: %s", key, strings.Join(supportedGlobalMappingAdditionalParams, ", "))
		}
		seen[normalizedKey] = struct{}{}
	}
	return envs, nil
}

func newSmbGlobalMappingCmd() string {
	return `$PWord = ConvertTo-SecureString -String $Env:smbpassword -AsPlainText -Force` +
		`;` +
		`$Credential = New-Object -TypeName System.Management.Automation.PSCredential -ArgumentList $Env:smbuser, $PWord` +
		`;` +
		`$Params = @{ RemotePath = $Env:smbremotepath; Credential = $Credential }` +
		`;` +
		`function HasValue([string]$Value) { return -not [string]::IsNullOrEmpty($Value) }` +
		`;if (HasValue $Env:smbopt_persistent) { $Params.Persistent = [System.Convert]::ToBoolean($Env:smbopt_persistent) }` +
		`;if (HasValue $Env:smbopt_requireintegrity) { $Params.RequireIntegrity = [System.Convert]::ToBoolean($Env:smbopt_requireintegrity) }` +
		`;if (HasValue $Env:smbopt_requireprivacy) { $Params.RequirePrivacy = [System.Convert]::ToBoolean($Env:smbopt_requireprivacy) }` +
		`;if (HasValue $Env:smbopt_usewritethrough) { $Params.UseWriteThrough = [System.Convert]::ToBoolean($Env:smbopt_usewritethrough) }` +
		`;if (HasValue $Env:smbopt_fullaccess) { $Params.FullAccess = $Env:smbopt_fullaccess -split ';' }` +
		`;if (HasValue $Env:smbopt_denyaccess) { $Params.DenyAccess = $Env:smbopt_denyaccess -split ';' }` +
		`;if (HasValue $Env:smbopt_transporttype) { $Params.TransportType = $Env:smbopt_transporttype }` +
		`;if (HasValue $Env:smbopt_skipcertificatecheck) { $Params.SkipCertificateCheck = [System.Convert]::ToBoolean($Env:smbopt_skipcertificatecheck) }` +
		`;if (HasValue $Env:smbopt_compressnetworktraffic) { $Params.CompressNetworkTraffic = [System.Convert]::ToBoolean($Env:smbopt_compressnetworktraffic) }` +
		`;if (HasValue $Env:smbopt_blockntlm) { $Params.BlockNTLM = [System.Convert]::ToBoolean($Env:smbopt_blockntlm) }` +
		`;if (HasValue $Env:smbopt_tcpport) { $Params.TcpPort = [UInt16]::Parse($Env:smbopt_tcpport) }` +
		`;if (HasValue $Env:smbopt_quicport) { $Params.QuicPort = [UInt16]::Parse($Env:smbopt_quicport) }` +
		`;if (HasValue $Env:smbopt_rdmaport) { $Params.RdmaPort = [UInt16]::Parse($Env:smbopt_rdmaport) }` +
		`;New-SmbGlobalMapping @Params`
}

func NewSmbGlobalMapping(remotePath, username, password string, requirePrivacy bool, globalMappingAdditionalParams string) error {
	// use PowerShell Environment Variables to store user input string to prevent command line injection
	// https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_environment_variables?view=powershell-5.1
	optionEnvs, err := parseGlobalMappingAdditionalParams(requirePrivacy, globalMappingAdditionalParams)
	if err != nil {
		return err
	}
	cmdLine := newSmbGlobalMappingCmd()
	envs := []string{
		fmt.Sprintf("smbuser=%s", username),
		fmt.Sprintf("smbpassword=%s", password),
		fmt.Sprintf("smbremotepath=%s", remotePath),
	}
	envs = append(envs, optionEnvs...)

	klog.V(2).Infof("begin to run NewSmbGlobalMapping with %s, %s, requirePrivacy=%v, globalMappingAdditionalParams=%q", remotePath, username, requirePrivacy, globalMappingAdditionalParams)
	if output, err := util.RunPowershellCmd(cmdLine, envs...); err != nil {
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
