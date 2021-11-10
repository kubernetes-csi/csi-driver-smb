## Driver Parameters
> Bring your own Samba server before using this driver.
### Storage Class Usage
> get an [example](../deploy/example/storageclass-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
csi.storage.k8s.io/provisioner-secret-name | secret name that stores `username`, `password`(`domain` is optional); if secret is provided, driver will create a sub directory with PV name under `source` | existing secret name |  No  |
csi.storage.k8s.io/provisioner-secret-namespace | namespace where the secret is | existing secret namespace |  No  |
csi.storage.k8s.io/node-stage-secret-name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
csi.storage.k8s.io/node-stage-secret-namespace | namespace where the secret is | existing secret namespace   |  Yes  |

### PV/PVC Usage
> get an [example](../deploy/example/pv-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
nodeStageSecretRef.name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  Yes  |
