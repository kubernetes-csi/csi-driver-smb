## Known issues

#### 1. mount error on Windows

 - error details:
```
FailedMount: MountVolume.MountDevice failed for volume "pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a" : rpc error: code = Internal desc = volume(pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a) mount "\\\\40.64.101.43\\share" on "\\var\\lib\\kubelet\\plugins\\kubernetes.io\\csi\\pv\\pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a\\globalmount" failed with smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed. output: "New-SmbGlobalMapping : A specified logon session does not exist. It may already have been terminated. \r\nAt line:1 char:190\r\n+ ... , $PWord;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Cred ...\r\n+   
```

 - Workaround
 To mount a SMB share on Windows, domain name should be provided in account, could use a dummy domain name, e.g. `Domain\AccountName`

#### 2. mount error on Windows after reboot

 - error details:
```
MountVolume.MountDevice failed for volume "pvc-1efb71f1-ab8a-4bbf-8db7-84a8e58877b4" : rpc error: code = Internal desc = volume(pvc-1efb71f1-ab8a-4bbf-8db7-84a8e58877b4) mount "//docp-smb1/smbservice" on "\var\lib\kubelet\plugins\kubernetes.io\csi\pv\pvc-1efb71f1-ab8a-4bbf-8db7-84a8e58877b4\globalmount" failed with smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed. output: "New-SmbGlobalMapping : Multiple connections to a server or shared resource by the same user, using more than one user \r\nname, are not allowed. Disconnect all previous connections to the server or shared resource and try again. \r\nAt line:1 char:190\r\n+ ... , $PWord;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Cred ...\r\n+ ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\r\n + CategoryInfo : NotSpecified: (MSFT_SmbGlobalMapping:ROOT/Microsoft/...mbGlobalMapping) [New-SmbGlobalMa \r\n pping], CimException\r\n + FullyQualifiedErrorId : Windows System Error 1219,New-SmbGlobalMapping\r\n \r\n", err: exit status 1
```

 - Workaround

**Only** use root share for one SMB server in one cluster and use `subPath` in deployment, if there is already `\\smb-server\share\test1` mounted, would get error when mounting volume `\\smb-server\share\test2` after Windows node reboot. Workaround is only use `\\smb-server\share` as `source`.

I will add notion in the windows example doc, there is no fix for this issue currently. Thanks.

 - Mitigation if hit `Multiple connections to a server or shared resource by the same user` error

log on to the Windows node, run `Get-SmbGlobalMapping` to list all mappings, run `Remove-SmbGlobalMapping -RemotePath xxx` to remove existing mapping, after a while, pod remount would succeed automatically
