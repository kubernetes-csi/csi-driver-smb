## Driver Parameters
> Bring your own Samba server before using this driver.
### Storage Class Usage
> get an [example](../deploy/example/storageclass-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
subDir | sub directory under smb share |  | No | if sub directory does not exist, this driver would create a new one
onDelete | when volume is deleted, keep the directory if it's `retain` | `delete`(default), `retain`, `archive`  | No | `delete`
csi.storage.k8s.io/provisioner-secret-name | secret name that stores `username`, `password`(`domain` is optional); if secret is provided, driver will create a sub directory with PV name under `source` | existing secret name |  No  |
csi.storage.k8s.io/provisioner-secret-namespace | namespace where the secret is | existing secret namespace |  No  |
csi.storage.k8s.io/node-stage-secret-name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
csi.storage.k8s.io/node-stage-secret-namespace | namespace where the secret is | existing secret namespace   |  Yes  |

 - VolumeID(`volumeHandle`) is the identifier of the volume handled by the driver, format of VolumeID: 
```
{smb-server-address}#{sub-dir-name}#{share-name}
```
> example: `smb-server.default.svc.cluster.local/share#subdir#`

### PV/PVC Usage
> get an [example](../deploy/example/pv-smb.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeHandle | Specify a value the driver can use to uniquely identify the share in the cluster. | A recommended way to produce a unique value is to combine the smb-server address, sub directory name and share name: `{smb-server-address}#{sub-dir-name}#{share-name}`. | Yes |
volumeAttributes.source | Samba Server address | `//smb-server-address/sharename` </br>([Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) format: `//accountname.file.core.windows.net/filesharename`) | Yes |
volumeAttributes.subDir | existing sub directory under smb share |  | No | sub directory must exist otherwise mount would fail
nodeStageSecretRef.name | secret name that stores `username`, `password`(`domain` is optional) | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  Yes  |

 - Use `kubectl create secret` to create `smbcreds` secret to store Samba Server username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```

### Kerberos ticket support for Linux
#### These are the conditions that must be met:
 - Kerberos support should be set up and cifs-utils must be installed on every node.
 - The directory /var/lib/kubelet/kerberos/ needs to exist, and it will hold kerberos credential cache files for various users.
 - This directory is shared between the host and the smb container.
 - The kerberos cache files are created for each volume and cleaned up during UnstageVolume phase
 - Each node should know to look up in that directory, here's example script for that, expected to be run on node provision:
```console
mkdir -p /etc/krb5.conf.d/
echo "[libdefaults]
default_ccache_name = FILE:/var/lib/kubelet/kerberos/krb5cc_%{uid}" > /etc/krb5.conf.d/ccache.conf
   ```
 - Mount flags should include **sec=krb5,uid=1000,cruid=1000**
   - sec=krb5 enables using credential cache
   - cruid=1000 provides information for what user credential cache will be looked up. This should match the secret entry.
   - uid=1000 is the owner of mounted files. This doesn't have to be the same as cruid.

#### Pass kerberos ticket in kubernetes secret 
To pass a ticket through secret, it needs to be acquired. Here's example how it can be done:

```console
export KRB5CCNAME="/var/lib/kubelet/kerberos/krb5cc_1000"
kinit USERNAME # Log in into domain
kvno cifs/lowercase_server_name # Acquire ticket for the needed share, it'll be written to the cache file
CCACHE=$(base64 -w 0 $KRB5CCNAME) # Get Base64-encoded cache
```

And passing the actual ticket to the secret, instead of the password.
Note that key for the ticket has included credential id, that must match exactly `cruid=` mount flag.
In theory, nothing prevents from having more than single ticket cache in the same secret.
```console
kubectl create secret generic smbcreds-krb5 --from-literal krb5cc_1000=$CCACHE
```

> See example of the [StorageClass](../deploy/example/storageclass-smb-krb5.yaml)

### Tips
#### `subDir` parameter supports following pv/pvc metadata conversion
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
