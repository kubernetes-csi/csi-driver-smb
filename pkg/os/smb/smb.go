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
	"strings"

	util "github.com/kubernetes-csi/csi-driver-smb/pkg/csi-common"
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

func NewSmbGlobalMapping(remotePath, username, password string) error {
	// use PowerShell Environment Variables to store user input string to prevent command line injection
	// https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_environment_variables?view=powershell-5.1
	cmdLine := fmt.Sprintf(`$PWord = ConvertTo-SecureString -String $Env:smbpassword -AsPlainText -Force` +
		`;$Credential = New-Object -TypeName System.Management.Automation.PSCredential -ArgumentList $Env:smbuser, $PWord` +
		`;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Credential $Credential -RequirePrivacy $true`)

	klog.V(2).Infof("begin to run NewSmbGlobalMapping with %s, %s", remotePath, username)
	if output, err := util.RunPowershellCmd(cmdLine, fmt.Sprintf("smbuser=%s", username),
		fmt.Sprintf("smbpassword=%s", password),
		fmt.Sprintf("smbremotepath=%s", remotePath)); err != nil {
		return fmt.Errorf("NewSmbGlobalMapping failed. output: %q, err: %v", string(output), err)
	}
	return nil
}

func RemoveSmbGlobalMapping(remotePath string) error {
	cmd := `Remove-SmbGlobalMapping -RemotePath $Env:smbremotepath -Force`
	klog.V(2).Infof("begin to run RemoveSmbGlobalMapping with %s", remotePath)
	if output, err := util.RunPowershellCmd(cmd, fmt.Sprintf("smbremotepath=%s", remotePath)); err != nil {
		return fmt.Errorf("UnmountSmbShare failed. output: %q, err: %v", string(output), err)
	}
	return nil
}
