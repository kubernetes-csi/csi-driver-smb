## CSI driver E2E usage example
### Prerequisite
 - [Set up a Samba Server on a Kubernetes cluster](./smb-provisioner/)
 > this example will create a new Samba Server(`//smb-server.default.svc.cluster.local/share`) with credential stored in secret `smbcreds`
 - Use `kubectl create secret` to create `smbcreds` secret to store Samba Server username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```
> add `--from-literal domain=DOMAIN-NAME` for domain support

### Option#1: Storage Class Usage
#### 1. Create a storage class
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
provisioner: smb.csi.k8s.io
parameters:
  source: "//smb-server.default.svc.cluster.local/share"
  csi.storage.k8s.io/node-stage-secret-name: "smbcreds"
  csi.storage.k8s.io/node-stage-secret-namespace: "default"
  createSubDir: "true"  # optional: create a sub dir for new volume
reclaimPolicy: Retain  # only retain is supported
volumeBindingMode: Immediate
mountOptions:
  - dir_mode=0777
  - file_mode=0777
  - uid=1001
  - gid=1001
```
 - Run below command to create a storage class
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/storageclass-smb.yaml
```

#### 2. Create a statefulset pod
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/statefulset.yaml
```
 - enter the pod container to verify
```console
# k exec -it statefulset-smb2-0 bash
root@statefulset-smb2-0:/# df -h
Filesystem                                    Size  Used Avail Use% Mounted on
...
//smb-server.default.svc.cluster.local/share  124G   15G  110G  12% /mnt/smb
/dev/sda1                                     124G   15G  110G  12% /etc/hosts
...
```

### Option#2: PV/PVC Usage
#### 1. Create PV/PVC bound with SMB share
 - Create a smb CSI PV, download [`pv-smb.yaml`](https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pv-smb.yaml) file and edit `source` in `volumeAttributes`
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-smb
spec:
  capacity:
    storage: 100Gi
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  mountOptions:
    - dir_mode=0777
    - file_mode=0777
    - vers=3.0
  csi:
    driver: smb.csi.k8s.io
    readOnly: false
    volumeHandle: unique-volumeid  # make sure it's a unique id in the cluster
    volumeAttributes:
      source: "//smb-server-address/sharename"
    nodeStageSecretRef:
      name: smbcreds
      namespace: default
```
> For [Azure File](https://docs.microsoft.com/en-us/azure/storage/files/), format of `source`: `//accountname.file.core.windows.net/sharename`

```console
kubectl create -f pv-smb.yaml
```

 - Create a PVC
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pvc-smb-static.yaml
```
 - make sure pvc is created and in `Bound` status finally
```console
watch kubectl describe pvc pvc-smb
```

#### 2.1 Create an deployment on Linux
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/deployment.yaml
```
 - enter the pod container to verify
```console
$ watch kubectl describe po nginx-smb
$ kubectl exec -it nginx-smb -- bash
# df -h
Filesystem            Size  Used Avail Use% Mounted on
...
/dev/sda1              97G   21G   77G  22% /etc/hosts
//20.43.191.64/share   97G   21G   77G  22% /mnt/smb
...
```
In the above example, there is a `/mnt/smb` directory mounted as cifs filesystem.

### 2.2 Create a deployment on Windows
```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/deployment.yaml
```
