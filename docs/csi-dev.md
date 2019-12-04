# Azure file CSI driver development guide

## How to build this project
 - Clone repo
```
$ mkdir -p $GOPATH/src/sigs.k8s.io/
$ git clone https://github.com/kubernetes-sigs/azurefile-csi-driver $GOPATH/src/sigs.k8s.io/azurefile-csi-driver
```

 - Build CSI driver
```
$ cd $GOPATH/src/sigs.k8s.io/azurefile-csi-driver
$ make azurefile
```

 - Run unit test
```
$ make unit-test
```

## How to test CSI driver in local environment

Install `csc` tool according to https://github.com/rexray/gocsi/tree/master/csc
```
$ mkdir -p $GOPATH/src/github.com
$ cd $GOPATH/src/github.com
$ git clone https://github.com/rexray/gocsi.git
$ cd rexray/gocsi/csc
$ make build
```

#### Start CSI driver locally
```
$ cd $GOPATH/src/sigs.k8s.io/azurefile-csi-driver
$ ./_output/azurefileplugin --endpoint tcp://127.0.0.1:10000 --nodeid CSINode -v=5 &
```
> Before running CSI driver, create "/etc/kubernetes/azure.json" file under testing server(it's better copy `azure.json` file from a k8s cluster with service principle configured correctly) and set `AZURE_CREDENTIAL_FILE` as following:
```
export set AZURE_CREDENTIAL_FILE=/etc/kubernetes/azure.json
```

#### 1. Get plugin info
```
$ csc identity plugin-info --endpoint tcp://127.0.0.1:10000
"file.csi.azure.com"    "v0.4.0"
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
$ csc node get-info --endpoint tcp://127.0.0.1:10000
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


## How to test CSI driver in a Kubernetes cluster

 - Build continer image and push image to dockerhub
```
# run `docker login` first
export REGISTRY=<dockerhub-alias>
make azurefile-container
make push-latest
```

 - Replace `mcr.microsoft.com/k8s/csi/azurefile-csi:latest` in `csi-azurefile-controller.yaml` and `csi-azurefile-node.yaml` with above dockerhub image urls and then follow [install CSI driver master version](https://github.com/kubernetes-sigs/azurefile-csi-driver/blob/master/docs/install-csi-driver-master.md)
 ```
wget -O csi-azurefile-controller.yaml https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/csi-azurefile-controller.yaml
# edit csi-azurefile-controller.yaml
kubectl apply -f csi-azurefile-controller.yaml

wget -O csi-azurefile-node.yaml https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/csi-azurefile-node.yaml
# edit csi-azurefile-node.yaml
kubectl apply -f csi-azurefile-node.yaml
 ```
