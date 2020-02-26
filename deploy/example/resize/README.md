# Volume Resizing Example

Azure File Driver supports both the offline and online scenario. 

## Enable Volume Resize Feature Gate

> Resize Feature Status
> Kubernetes 1.14, 1.15: alpha
> Kubernetes 1.16+:  beta

In Kubernetes 1.14 and 1.15, CSI volume resizing is still alpha. So the following feature gate is needed to be enabled.

```
--feature-gates=ExpandCSIVolumes=true
```

In Kuberntest 1.16+, the feature has been beta. The feature gate is enabled by default.

## Example

1. Set `allowVolumeExpansion` field as true in the storageclass manifest.  

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: file.csi.azure.com
provisioner: file.csi.azure.com
allowVolumeExpansion: true
parameters:
  skuName: Standard_LRS
reclaimPolicy: Delete
volumeBindingMode: Immediate
```

2. Create storageclass, pvc and pod.

```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/storageclass-azurefile-csi.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/nginx-pod-azurefile.yaml
```

3. Check the PV size
```console
$ kubectl get pvc pvc-azurefile
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
pvc-azurefile   Bound    pvc-74dc3e29-534f-4d54-98fc-731adb46c948   15Gi       RWX            file.csi.azure.com   57m
```
4. Check the filesystem size in the container.

```console
$ kubectl exec -it nginx-azurefile -- df -h /mnt/azurefile
Filesystem                                                                                Size  Used Avail Use% Mounted on
//fuse0575b5cff3b641d7a0c.file.core.windows.net/pvc-74dc3e29-534f-4d54-98fc-731adb46c948   15G  128K   15G   1% /mnt/azurefile
```

4. Expand the pvc by increasing the field `spec.resources.requests.storage`.

```console
$ kubectl edit pvc pvc-azurefile
...
...
spec:
  resources:
    requests:
      storage: 20Gi
...
...
```

5. Verify the filesystem size.

```console
$ kubectl get pvc pvc-azurefile
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS         AGE
pvc-azurefile   Bound    pvc-74dc3e29-534f-4d54-98fc-731adb46c948   20Gi       RWX            file.csi.azure.com   65m

$ kubectl exec -it nginx-azurefile -- df -h /mnt/azurefile
Filesystem                                                                                Size  Used Avail Use% Mounted on
//fuse0575b5cff3b641d7a0c.file.core.windows.net/pvc-74dc3e29-534f-4d54-98fc-731adb46c948   20G  128K   20G   1% /mnt/azurefile
```
