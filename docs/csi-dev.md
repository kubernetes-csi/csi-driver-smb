# Azure file CSI driver

## Usage:

### Build azurefileplugin
```
$ make azurefile
```

### Start CSI driver
```
$ sudo ./_output/azurefileplugin --endpoint tcp://127.0.0.1:10000 --nodeid CSINode -v=5
```

### Test using csc
Get ```csc``` tool from https://github.com/rexray/gocsi/tree/master/csc

#### Get plugin info
```
$ csc identity plugin-info --endpoint tcp://127.0.0.1:10000
"csi-azurefile" "v0.5.0-alpha"
```

#### Before create a volume, create "/etc/kubernetes/azure.json" file under testing server

#### Create a volume
```
$ csc controller new --endpoint tcp://127.0.0.1:10000 --cap 1,block CSIVolumeName  --req-bytes 2147483648 --params skuname=Standard_LRS
CSIVolumeID       2147483648      "accountname"="f5713de20cde511e8ba4900" "skuname"="Standard_LRS"
```

#### NodePublish a volume
```
$ mkdir ~/testmount
$ csc node publish --endpoint tcp://127.0.0.1:10000 --cap 1,block --target-path ~/testmount CSIVolumeID
#f5713de20cde511e8ba4900#pvc-file-dynamic-8ff5d05a-f47c-11e8-9c3a-000d3a00df41
```

#### NodeUnpublish a volume
```
$ csc node unpublish --endpoint tcp://127.0.0.1:10000 --target-path ~/testmount CSIVolumeID
CSIVolumeID
```

#### Delete a volume
```
$ csc controller del --endpoint tcp://127.0.0.1:10000 CSIVolumeID
CSIVolumeID
```

#### Validate volume capabilities
```
$ csc controller validate-volume-capabilities --endpoint tcp://127.0.0.1:10000 --cap 1,block CSIVolumeName
CSIVolumeID  true
```

#### Get NodeID
```
$ csc node get-id --endpoint tcp://127.0.0.1:10000
CSINode
```
