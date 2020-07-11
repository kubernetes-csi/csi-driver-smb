## `smb.csi.k8s.io` driver parameters
> This driver only supports static provisioning, bring your own Samba server before using this driver.
### Storage Class Usage
> get a quick example [here](../deploy/example/storageclass-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
source | Samba Server address | `//smb-server-address/sharename`(for [Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction), format is `//accountname.file.core.windows.net/filesharename`) | Yes |
csi.storage.k8s.io/node-stage-secret-name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
csi.storage.k8s.io/node-stage-secret-namespace | namespace where the secret is | k8s namespace  |  Yes  |

### PV/PVC Usage
> get a quick example [here](../deploy/example/pv-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.source | Samba Server address | `//smb-server-address/sharename`(for [Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction), format is `//accountname.file.core.windows.net/filesharename`) | Yes |
nodeStageSecretRef.name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  Yes  |
