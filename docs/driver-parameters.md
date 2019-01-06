## `file.csi.azure.com` driver parameters
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

 - Dynamic Provisioning
  > get a quick example [here](../deploy/example/storageclass-azurefile-csi.yaml)

Name | Meaning | Example | Mandatory | Default value 
--- | --- | --- | --- | ---
skuName | azure file storage account type (alias: `storageAccountType`) | `Standard_LRS`, `Standard_GRS`, `Standard_RAGRS` | No | `Standard_LRS`
storageAccount | specify the storage account name in which azure file share will be created | STORAGE_ACCOUNT_NAME | No | if empty, driver will find a suitable storage account that matches `skuName` in the same resource group
location | specify the location in which azure file share will be created | `eastus`, `westus`, etc. | No | if empty, driver will use the same location name as current k8s cluster
resourceGroup | specify the resource group in which azure file share will be created | RG_NAME | No | if empty, driver will use the same resource group name as current k8s cluster

 - Static Provisioning(use existing azure disk)
  > get a quick example [here](../deploy/example/pv-azurefile-csi.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.sharename | azure file share name | existing azure file share name | Yes |
nodePublishSecretRef.name | secret name that stores storage account name and key | existing secret name |  Yes  | 
nodePublishSecretRef.namespace | namespace where the secret is | k8s namespace  |  No  | `default`
