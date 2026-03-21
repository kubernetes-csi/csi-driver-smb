# CSI Driver E2E Usage Example

> Refer to [driver parameters](../../docs/driver-parameters.md) for more detailed usage.

## Prerequisites

- [Set up a Samba Server on a Kubernetes cluster](./smb-provisioner/)
  > This example will create a new Samba Server (`//smb-server.default.svc.cluster.local/share`) with credentials stored in the secret `smbcreds`.

- Use `kubectl create secret` to create the `smbcreds` secret to store Samba Server username and password:

  ```console
  kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
  ```

  > Add `--from-literal domain=DOMAIN-NAME` for domain support.

---

## Option 1: Storage Class Usage (Dynamic Provisioning)

### 1. Create a StorageClass

#### Access by Linux node

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
provisioner: smb.csi.k8s.io
parameters:
  source: //smb-server.default.svc.cluster.local/share
  # if csi.storage.k8s.io/provisioner-secret is provided, will create a sub directory
  # with PV name under source
  csi.storage.k8s.io/provisioner-secret-name: smbcreds
  csi.storage.k8s.io/provisioner-secret-namespace: default
  csi.storage.k8s.io/node-stage-secret-name: smbcreds
  csi.storage.k8s.io/node-stage-secret-namespace: default
reclaimPolicy: Delete  # available values: Delete, Retain
volumeBindingMode: Immediate
allowVolumeExpansion: true
mountOptions:
  - dir_mode=0777
  - file_mode=0777
  - uid=1001
  - gid=1001
```

```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/storageclass-smb.yaml
```

#### Access by Windows node

> **Note:** `smb-server.default.svc.cluster.local` cannot be resolved by CSI proxy on Windows nodes. Use a public IP address or externally resolvable domain name for `source` in the StorageClass.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
provisioner: smb.csi.k8s.io
parameters:
  # On Windows, "*.default.svc.cluster.local" cannot be resolved by csi-proxy.
  # Replace with a public IP or externally resolvable domain name.
  source: //<public-ip-or-domain>/share
  # if csi.storage.k8s.io/provisioner-secret is provided, will create a sub directory
  # with PV name under source
  csi.storage.k8s.io/provisioner-secret-name: smbcreds
  csi.storage.k8s.io/provisioner-secret-namespace: default
  csi.storage.k8s.io/node-stage-secret-name: smbcreds
  csi.storage.k8s.io/node-stage-secret-namespace: default
volumeBindingMode: Immediate
allowVolumeExpansion: true
mountOptions:
  - dir_mode=0777
  - file_mode=0777
```

### 2. Create a StatefulSet with SMB volume mount

```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/statefulset.yaml
```

Verify the volume is mounted:

```console
kubectl exec -it statefulset-smb-0 -- df -h
```

<pre>
Filesystem                                    Size  Used Avail Use% Mounted on
...
//smb-server.default.svc.cluster.local/share  124G   23G  102G  19% /mnt/smb
/dev/sda1                                     124G   15G  110G  12% /etc/hosts
...
</pre>

---

## Option 2: PV/PVC Usage (Static Provisioning)

### 1. Create PV/PVC bound with SMB share

Create an SMB CSI PV — download [`pv-smb.yaml`](https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pv-smb.yaml) and edit `source` in `volumeAttributes`:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  annotations:
    pv.kubernetes.io/provisioned-by: smb.csi.k8s.io
  name: pv-smb
spec:
  capacity:
    storage: 100Gi
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: smb
  mountOptions:
    - dir_mode=0777
    - file_mode=0777
  csi:
    driver: smb.csi.k8s.io
    # volumeHandle format: {smb-server-address}#{sub-dir-name}#{share-name}
    # make sure this value is unique for every share in the cluster
    volumeHandle: smb-server.default.svc.cluster.local/share##
    volumeAttributes:
      source: //smb-server-address/sharename
    nodeStageSecretRef:
      name: smbcreds
      namespace: default
```

> For [Azure File](https://learn.microsoft.com/en-us/azure/storage/files/), the format of `source` is: `//accountname.file.core.windows.net/sharename`

```console
kubectl create -f pv-smb.yaml
```

Create a PVC:

```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pvc-smb-static.yaml
```

Make sure the PVC is created and in `Bound` status:

```console
watch kubectl describe pvc pvc-smb
```

### 2. Create a deployment

#### Linux

```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/deployment.yaml
```

Verify the volume is mounted:

```console
kubectl exec -it nginx-smb -- df -h
```

<pre>
Filesystem            Size  Used Avail Use% Mounted on
...
/dev/sda1              97G   21G   77G  22% /etc/hosts
//20.43.191.64/share   97G   21G   77G  22% /mnt/smb
...
</pre>

In the above example, `/mnt/smb` is mounted as a CIFS filesystem.

#### Windows

```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/deployment.yaml
```
