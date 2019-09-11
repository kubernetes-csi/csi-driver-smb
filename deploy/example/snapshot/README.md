# Snapshot Example

> Attention: Since volume snapshot is an alpha feature in Kubernetes currently, you need to enable a new alpha feature gate called `VolumeSnapshotDataSource` in the Kubernetes master.
>
> ```
> --feature-gates=VolumeSnapshotDataSource=true
> ```

## Create a StorageClass

```console
kubectl apply -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/storageclass-azurefile-csi.yaml
```

## Create a PVC

```console
kubectl apply -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/pvc-azurefile-csi.yaml
```

## Create a VolumeSnapshotClass

```console
kubectl apply -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/snapshot/volumesnapshotclass-azurefile.yaml
```

## Create a VolumeSnapshot

```console
kubectl apply -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/snapshot/volumesnapshot-azurefile.yaml
```

## Delete a VolumeSnapshot

```console
kubectl delete -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/snapshot/volumesnapshot-azurefile.yaml
```

## Delete a VolumeSnapshotClass

```console
kubectl delete -f $GOPATH/src/github.com/kubernetes-sigs/azurefile-csi-driver/deploy/example/snapshot/volumesnapshotclass-azurefile.yaml
```
