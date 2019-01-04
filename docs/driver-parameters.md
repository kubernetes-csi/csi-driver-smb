## `file.csi.azure.com` driver parameters
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

Name | Meaning | Example | Mandatory | Default value 
--- | --- | --- | --- | ---
skuName | azure file storage account type | `Standard_LRS`, `Standard_GRS`, `Standard_RAGRS` | No | `Standard_LRS`)
storageAccount | specify the storage account name in which azure file share will be created | STORAGE_ACCOUNT_NAME | No | if empty, driver will find a suitable storage account that matches `skuName` in the same resource group
location | specify the location in which azure file share will be created | `eastus`, `westus`, etc. | No | if empty, driver will use the same location name as current k8s cluster
resourceGroup | specify the resource group in which azure file share will be created | RG_NAME | No | if empty, driver will use the same resource group name as current k8s cluster

