## `smb.csi.k8s.io` driver parameters
 > storage class `smb.csi.k8s.io` parameters are compatible with built-in [smb](https://kubernetes.io/docs/concepts/storage/volumes/#smb) plugin

 - Dynamic Provision
  > get a quick example [here](../deploy/example/storageclass-smb-csi.yaml)

Name | Meaning | Example | Mandatory | Default value 
--- | --- | --- | --- | ---
skuName | smb storage account type (alias: `storageAccountType`) | `Standard_LRS`, `Standard_ZRS`, `Standard_GRS`, `Standard_RAGRS`, `Premium_LRS` | No | `Standard_LRS` <br><br> Note:  <br> 1. minimum file share size of Premium account type is `100GB`<br> 2.[`ZRS` account type](https://docs.microsoft.com/en-us/azure/storage/common/storage-redundancy#zone-redundant-storage) is supported in limited regions <br> 3. Premium files shares is currently only available for LRS
storageAccount | specify the storage account name in which smb share will be created | STORAGE_ACCOUNT_NAME | No | if empty, driver will find a suitable storage account that matches `skuName` in the same resource group; if a storage account name is provided, it means that storage account must exist otherwise there would be error
location | specify the location in which smb share will be created | `eastus`, `westus`, etc. | No | if empty, driver will use the same location name as current k8s cluster
resourceGroup | specify the resource group in which smb share will be created | existing resource group name | No | if empty, driver will use the same resource group name as current k8s cluster
shareName | specify smb share name | existing or new smb name | No | if empty, driver will generate an smb share name
storeAccountKey | whether store account key to k8s secret | `true`,`false` | No | `true`
secretNamespace | specify the namespace of secret to store account key | `default`,`kube-system`,etc | No | `default`
--- | following parameters are only for [VHD disk feature](../deploy/example/disk) | --- | --- |
fsType | File System Type | `ext4`, `ext3`, `ext2`, `xfs` | Yes | `ext4`
diskName | existing VHD disk file name | `pvc-062196a6-6436-11ea-ab51-9efb888c0afb.vhd` | No |

 - Static Provision(use existing smb)
  > get a quick example [here](../deploy/example/pv-smb-csi.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.sharename | smb share name | existing smb share name | Yes |
nodeStageSecretRef.name | secret name that stores storage account name and key | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  No  | `default`
