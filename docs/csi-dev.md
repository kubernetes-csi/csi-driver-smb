# Azure file CSI driver development guide

### Build azurefileplugin
```
$ make azurefile
```

> Before running CSI driver, create "/etc/kubernetes/azure.json" file under testing server(it's better copy `azure.json` file from a k8s cluster with service principle configured correctly) and set `AZURE_CREDENTIAL_FILE` as following:
```
export set AZURE_CREDENTIAL_FILE=/etc/kubernetes/azure.json
```

### Start CSI driver
```
$ ./_output/azurefileplugin --endpoint tcp://127.0.0.1:10000 --nodeid CSINode -v=5
```

### Test using csc
Get ```csc``` tool from https://github.com/rexray/gocsi/tree/master/csc

#### 1. Get plugin info
```
$ csc identity plugin-info --endpoint tcp://127.0.0.1:10000
"csi-azurefile" "v0.5.0-alpha"
```

#### 2. Create an azure file volume
```
$ csc controller new --endpoint tcp://127.0.0.1:10000 --cap 1,block CSIVolumeName  --req-bytes 2147483648 --params skuname=Standard_LRS
CSIVolumeID       2147483648      "accountname"="f5713de20cde511e8ba4900" "skuname"="Standard_LRS"
```

#### 3. Mount an azure file volume to a user specified directory
```
$ mkdir ~/testmount
$ csc node publish --endpoint tcp://127.0.0.1:10000 --cap 1,block --target-path ~/testmount CSIVolumeID
#f5713de20cde511e8ba4900#pvc-file-dynamic-8ff5d05a-f47c-11e8-9c3a-000d3a00df41
```

#### 4. Unmount azure file volume
```
$ csc node unpublish --endpoint tcp://127.0.0.1:10000 --target-path ~/testmount CSIVolumeID
CSIVolumeID
```

#### 5. Delete azure file volume
```
$ csc controller del --endpoint tcp://127.0.0.1:10000 CSIVolumeID
CSIVolumeID
```

#### 6. Validate volume capabilities
```
$ csc controller validate-volume-capabilities --endpoint tcp://127.0.0.1:10000 --cap 1,block CSIVolumeID
CSIVolumeID  true
```

#### 7. Get NodeID
```
$ csc node get-id --endpoint tcp://127.0.0.1:10000
CSINode
```

#### 8. Create snapshot
```
$  csc controller create-snapshot
```

#### 9. Delete snapshot
```
$  csc controller delete-snapshot
```
