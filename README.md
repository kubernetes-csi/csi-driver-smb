# azurefile CSI driver for Kubernetes (Alpha)
![TravisCI](https://travis-ci.com/andyzhangx/azurefile-csi-driver.svg?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fandyzhangx%2Fazurefile-csi-driver.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fandyzhangx%2Fazurefile-csi-driver?ref=badge_shield)

**WARNING**: This driver is in ALPHA currently. Do NOT use this driver in a production environment in its current state.

 - supported Kubernetes version: v1.12.0 or later version
 - supported agent OS: Linux

### About
This driver allows Kubernetes to use [azure file](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) volume, csi plugin name: `file.csi.azure.com`

### Driver parameters
Please refer to [`file.csi.azure.com` driver parameters](./docs/driver-parameters.md)
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

## Prerequisite
 - To ensure that all necessary features are enabled, set the following feature gate flags to true:
```
--feature-gates=CSIPersistentVolume=true,MountPropagation=true,VolumeSnapshotDataSource=true,KubeletPluginsWatcher=true,CSINodeInfo=true,CSIDriverRegistry=true
```
CSIPersistentVolume is enabled by default in v1.10. MountPropagation is enabled by default in v1.10. VolumeSnapshotDataSource is a new alpha feature in v1.12. KubeletPluginsWatcher is enabled by default in v1.12. CSINodeInfo and CSIDriverRegistry are new alpha features in v1.12.

 - An [Cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md) should already exist on all agent nodes
 > usually it's `/etc/kubernetes/azure.json` deployed by AKS or acs-engine, and supports both `service principal` and `msi`

### Install azurefile CSI driver on a kubernetes cluster
Please refer to [install azurefile csi driver](https://github.com/andyzhangx/azurefile-csi-driver/blob/master/docs/install-azurefile-csi-driver.md)

## Example
### 1. create a pod with csi azurefile driver mount on linux
#### Example#1: Azurefile Dynamic Provisioning
 - Create an azurefile CSI storage class
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/storageclass-azurefile-csi.yaml
```

 - Create an azurefile CSI PVC
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi.yaml
```

#### Example#2: Azurefile Static Provisioning(use an existing azure file share)
 - Use `kubectl create secret` to create `azure-secret` with existing storage account name and key
```
kubectl create secret generic azure-secret --from-literal accountname=NAME --from-literal accountkey="KEY" --type=Opaque
```

 - Create an azurefile CSI PV, download `pv-azurefile-csi.yaml` file and edit `sharename` in `volumeAttributes`
```
wget https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/pv-azurefile-csi.yaml
vi pv-azurefile-csi.yaml
kubectl create -f pv-azurefile-csi.yaml
```

 - Create an azurefile CSI PVC which would be bound to the above PV
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi-static.yaml
```

### 2. validate PVC status and create an nginx pod
 - make sure pvc is created and in `Bound` status finally
```
watch kubectl describe pvc pvc-azurefile
```

 - create a pod with azurefile CSI PVC
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/nginx-pod-azurefile.yaml
```

### 3. enter the pod container to do validation
 - watch the status of pod until its Status changed from `Pending` to `Running` and then enter the pod container
```
$ watch kubectl describe po nginx-azurefile
$ kubectl exec -it nginx-azurefile -- bash
root@nginx-azurefile:/# df -h
Filesystem                                                                                             Size  Used Avail Use% Mounted on
overlay                                                                                                 30G   19G   11G  65% /
tmpfs                                                                                                  3.5G     0  3.5G   0% /dev
...
//f571xxx.file.core.windows.net/pvc-file-dynamic-e2ade9f3-f88b-11e8-8429-000d3a03e7d7  1.0G   64K  1.0G   1% /mnt/azurefile
...
```
In the above example, there is a `/mnt/azurefile` directory mounted as dysk filesystem.

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)


### Links
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/Home.html)
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fandyzhangx%2Fazurefile-csi-driver.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fandyzhangx%2Fazurefile-csi-driver?ref=badge_large)