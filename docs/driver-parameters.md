## Driver Parameters
> Bring your own Samba server before using this driver.
### Storage Class Usage
> get an [example](../deploy/example/storageclass-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
subDir | sub directory under smb share |  | No | if sub directory does not exist, this driver would create a new one
csi.storage.k8s.io/provisioner-secret-name | secret name that stores `username`, `password`(`domain` is optional); if secret is provided, driver will create a sub directory with PV name under `source` | existing secret name |  No  |
csi.storage.k8s.io/provisioner-secret-namespace | namespace where the secret is | existing secret namespace |  No  |
csi.storage.k8s.io/node-stage-secret-name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
csi.storage.k8s.io/node-stage-secret-namespace | namespace where the secret is | existing secret namespace   |  Yes  |

### PV/PVC Usage
> get an [example](../deploy/example/pv-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
volumeAttributes.subDir | existing sub directory under smb share |  | No | sub directory must exist otherwise mount would fail
nodeStageSecretRef.name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  Yes  |

 - Use `kubectl create secret` to create `smbcreds` secret to store Samba Server username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```

### Tips
#### `subDir` parameter supports following pv/pvc metadata convension
> if `subDir` value contains following string, it would be converted into corresponding pv/pvc name or namespace
 - `${pvc.metadata.name}`
 - `${pvc.metadata.namespace}`
 - `${pv.metadata.name}`

#### provide `mountOptions` for `DeleteVolume`
> since `DeleteVolumeRequest` does not provide `mountOptions`, following is the workaround to provide `mountOptions` for `DeleteVolume`
  - create a secret `smbcreds` with `mountOptions`
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD" --from-literal mountOptions="dir_mode=0777,file_mode=0777,uid=0,gid=0,mfsymlinks"
```

 - set `csi.storage.k8s.io/provisioner-secret-name: "smbcreds"` in storage class
