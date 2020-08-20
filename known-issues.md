## Known issues

#### 1. mount error on Windows: `New-SmbGlobalMapping : A specified logon session does not exist`
```
FailedMount: MountVolume.MountDevice failed for volume "pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a" : rpc error: code = Internal desc = volume(pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a) mount "\\\\40.64.101.43\\share" on "\\var\\lib\\kubelet\\plugins\\kubernetes.io\\csi\\pv\\pvc-2ca92cca-c690-4fea-842f-0a4d32e97f5a\\globalmount" failed with smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed. output: "New-SmbGlobalMapping : A specified logon session does not exist. It may already have been terminated. \r\nAt line:1 char:190\r\n+ ... ser, $PWord;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Cred ...\r\n+   
```

 - Fix
 To mount a SMB share on Windows, domain name should be provided in account, could use a dummy domain name, e.g. `Domain\AccountName`
