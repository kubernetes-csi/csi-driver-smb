## CSI driver E2E usage example
### Prerequisite
 - [Set up a SMB server on a Kubernetes cluster](./smb-provisioner/)

#### 1. Create PV/PVC bound with SMB share
 - Use `kubectl create secret` to create `smbcreds` with SMB username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```
> add `--from-literal domain=DOMAIN-NAME` for domain support

 - Create a smb CSI PV, download [`pv-smb-csi.yaml`](https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pv-smb-csi.yaml) file and edit `source` in `volumeAttributes`
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
kubectl create -f pv-smb-csi.yaml
```

 - Create a PVC
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pvc-smb-csi-static.yaml
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
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/deployment.yaml
```