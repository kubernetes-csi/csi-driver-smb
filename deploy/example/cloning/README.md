# Volume Cloning Example

- supported from v1.11.0

## Create a Source PVC

```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/storageclass-smb.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pvc-smb.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/nginx-pod-smb.yaml
```

### Check the Source PVC

```console
$ kubectl exec nginx-smb -- ls /mnt/smb
outfile
```

## Create a PVC from an existing PVC
>  Make sure application is not writing data to source smb share
```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/cloning/pvc-smb-cloning.yaml
```
### Check the Creation Status

```console
$ kubectl describe pvc pvc-smb-cloning
Name:          pvc-smb-cloning
Namespace:     default
StorageClass:  smb
Status:        Bound
Volume:        pvc-e48e8ace-578f-4031-8e1e-9343e75c2c05
Labels:        <none>
Annotations:   pv.kubernetes.io/bind-completed: yes
               pv.kubernetes.io/bound-by-controller: yes
               volume.beta.kubernetes.io/storage-provisioner: smb.csi.k8s.io
               volume.kubernetes.io/storage-provisioner: smb.csi.k8s.io
Finalizers:    [kubernetes.io/pvc-protection]
Capacity:      10Gi
Access Modes:  RWX
VolumeMode:    Filesystem
DataSource:
  Kind:   PersistentVolumeClaim
  Name:   pvc-smb
Used By:  <none>
Events:
  Type    Reason                 Age   From                                                                                   Message
  ----    ------                 ----  ----                                                                                   -------
  Normal  ExternalProvisioning   12s   persistentvolume-controller                                                            waiting for a volume to be created, either by external provisioner "smb.csi.k8s.io" or manually created by system administrator
  Normal  Provisioning           12s   smb.csi.k8s.io_aks-nodepool1-34988195-vmss000001_7eccface-64d7-4084-9b9c-edebdd7a6855  External provisioner is provisioning volume for claim "default/pvc-smb-cloning"
  Normal  ProvisioningSucceeded  12s   smb.csi.k8s.io_aks-nodepool1-34988195-vmss000001_7eccface-64d7-4084-9b9c-edebdd7a6855  Successfully provisioned volume pvc-e48e8ace-578f-4031-8e1e-9343e75c2c05
```

## Restore the PVC into a Pod

```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/cloning/nginx-pod-restored-cloning.yaml
```

### Check Sample Data

```console
$ kubectl exec nginx-smb-restored-cloning -- ls /mnt/smb
outfile
```