## `file.csi.azure.com` driver parameters
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

 - Dynamic Provision
  > get a quick example [here](../deploy/example/storageclass-azurefile-csi.yaml)

Name | Meaning | Example | Mandatory | Default value 
--- | --- | --- | --- | ---
skuName | azure file storage account type (alias: `storageAccountType`) | `Standard_LRS`, `Standard_ZRS`, `Standard_GRS`, `Standard_RAGRS`, `Premium_LRS` | No | `Standard_LRS` <br><br> Note:  <br> 1. minimum file share size of Premium account type is `100GB`<br> 2.[`ZRS` account type](https://docs.microsoft.com/en-us/azure/storage/common/storage-redundancy#zone-redundant-storage) is supported in limited regions <br> 3. Premium files shares is currently only available for LRS
storageAccount | specify the storage account name in which azure file share will be created | STORAGE_ACCOUNT_NAME | No | if empty, driver will find a suitable storage account that matches `skuName` in the same resource group; if a storage account name is provided, it means that storage account must exist otherwise there would be error
location | specify the location in which azure file share will be created | `eastus`, `westus`, etc. | No | if empty, driver will use the same location name as current k8s cluster
resourceGroup | specify the resource group in which azure file share will be created | existing resource group name | No | if empty, driver will use the same resource group name as current k8s cluster
shareName | specify azure file share name | existing or new azure file name | No | if empty, driver will generate an azure file share name
--- | following parameters are only for [VHD disk feature](../deploy/example/disk) | --- | --- |
fsType | File System Type | `ext4`, `ext3`, `ext2`, `xfs` | Yes | `ext4`
diskName | existing VHD disk file name | `pvc-062196a6-6436-11ea-ab51-9efb888c0afb.vhd` | No |

 - Static Provision(use existing azure file)
  > get a quick example [here](../deploy/example/pv-azurefile-csi.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.sharename | azure file share name | existing azure file share name | Yes |
nodeStageSecretRef.name | secret name that stores storage account name and key | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  No  | `default`
