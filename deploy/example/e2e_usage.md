## CSI driver E2E usage example
#### 1. create a pod with smb share mount
##### Static Provisioning(use an existing smb share)
 - Use `kubectl create secret` to create `smbcreds` with SMB username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```
> add `--from-literal domain=DOMAINNAME` for domain support

 - Create an smb CSI PV, download [`pv-smb-csi.yaml`](https://raw.githubusercontent.com/csi-driver/csi-driver-smb/master/deploy/example/pv-smb-csi.yaml) file and edit `source` in `volumeAttributes`
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
    volumeHandle: arbitrary-volumeid  # make sure it's a unique id
    volumeAttributes:
      source: "//IP/smb-server/directory"
    nodeStageSecretRef:
      name: smbcreds
      namespace: default
```

```console
kubectl create -f pv-smb-csi.yaml
```

 - Create a PVC
```console
kubectl create -f https://raw.githubusercontent.com/csi-driver/csi-driver-smb/master/deploy/example/pvc-smb-csi-static.yaml
```

#### 2. validate PVC status and create an nginx pod
 - make sure pvc is created and in `Bound` status finally
```console
watch kubectl describe pvc pvc-smb
```

 - create a pod with smb CSI PVC
```console
kubectl create -f https://raw.githubusercontent.com/csi-driver/csi-driver-smb/master/deploy/example/nginx-pod-smb.yaml
```

#### 3. enter the pod container to do validation
 - watch the status of pod until its Status changed from `Pending` to `Running` and then enter the pod container
```console
$ watch kubectl describe po nginx-smb
$ kubectl exec -it nginx-smb -- bash
root@nginx-smb:/# df -h
Filesystem                                                                Size  Used Avail Use% Mounted on
overlay                                                                   30G   19G  11G   65%  /
tmpfs                                                                     3.5G  0    3.5G  0%   /dev
...
//f571xxx.file.core.windows.net/pvc-54caa11f-9e27-11e9-ba7b-0601775d3b69  1.0G  64K  1.0G  1%   /mnt/smb
...
```
In the above example, there is a `/mnt/smb` directory mounted as cifs filesystem.
