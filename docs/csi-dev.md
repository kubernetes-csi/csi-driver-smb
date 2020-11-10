# Azure file CSI driver development guide

## How to build this project
 - Clone repo
```console
$ mkdir -p $GOPATH/src/sigs.k8s.io/
$ git clone https://github.com/kubernetes-csi/csi-driver-smb $GOPATH/src/github.com/kubernetes-csi/csi-driver-smb
```

 - Build CSI driver
```console
$ cd $GOPATH/src/github.com/kubernetes-csi/csi-driver-smb
$ make
```

 - Run verification test before submitting code
```console
$ make verify
```

## How to test CSI driver in local environment

Install `csc` tool according to https://github.com/rexray/gocsi/tree/master/csc
```console
$ mkdir -p $GOPATH/src/github.com
$ cd $GOPATH/src/github.com
$ git clone https://github.com/rexray/gocsi.git
$ cd rexray/gocsi/csc
$ make build
```

#### Start CSI driver locally
```console
$ cd $GOPATH/src/github.com/kubernetes-csi/csi-driver-smb
$ ./_output/smbplugin --endpoint tcp://127.0.0.1:10000 --nodeid CSINode -v=5 &
```

#### 1. Get plugin info
```console
$ csc identity plugin-info --endpoint tcp://127.0.0.1:10000
"smb.csi.k8s.io"    "v0.1.0"
```

#### 2. Stage a SMB volume on a node
```console
$ csc node stage --endpoint tcp://127.0.0.1:10000 --cap 1,block --staging-target-path=/tmp/staging-path --with-requires-attribs ... --with-requires-creds ...
```

#### 3. Publish a SMB volume on a node (bind mount the volume from staging to target path)
```
$ csc node publish --endpoint tcp://127.0.0.1:10000 --cap 1,block --staging-target-path=/tmp/staging-path --target-path=/tmp/publish-path volumeid
```

#### 4. Unpublish a SMB volume on a node
```
$ csc node unpublish --endpoint tcp://127.0.0.1:10000 --target-path=/tmp/publish-path volumeid
```

#### 5. Unstage a SMB volume on a node
```
$ csc node unstage --endpoint tcp://127.0.0.1:10000 --staging-target-path=/tmp/staging-path volumeid
```

#### 6. Validate volume capabilities
```console
$ csc controller validate-volume-capabilities --endpoint tcp://127.0.0.1:10000 --cap 1,block CSIVolumeID
CSIVolumeID  true
```

#### 7. Get NodeID
```console
$ csc node get-info --endpoint tcp://127.0.0.1:10000
CSINode
```

## How to test CSI driver in a Kubernetes cluster

 - Build continer image and push image to dockerhub
```console
# run `docker login` first
export REGISTRY=<dockerhub-alias>
export IMAGE_VERSION=latest
# build linux, windows 1809, 1903, 1909, and 2004 images
make container-all
# create a manifest list for the images above
make push-manifest
```

 - Replace `mcr.microsoft.com/k8s/csi/smb-csi:latest` in [`csi-smb-controller.yaml`](https://github.com/kubernetes-csi/csi-driver-smb/blob/master/deploy/csi-smb-controller.yaml) and [`csi-smb-node.yaml`](https://github.com/kubernetes-csi/csi-driver-smb/blob/master/deploy/csi-smb-node.yaml) with above dockerhub image urls and then follow [install CSI driver master version](https://github.com/kubernetes-csi/csi-driver-smb/blob/master/docs/install-csi-driver-master.md)
 ```console
wget -O csi-smb-node.yaml https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/csi-smb-node.yaml
# edit csi-smb-node.yaml
kubectl apply -f csi-smb-node.yaml
 ```

### How to update chart index

```console
helm repo index charts --url=https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts
```
