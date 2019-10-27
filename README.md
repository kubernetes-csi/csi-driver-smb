# azurefile CSI driver for Kubernetes
[![Travis](https://travis-ci.org/kubernetes-sigs/azurefile-csi-driver.svg)](https://travis-ci.org/kubernetes-sigs/azurefile-csi-driver)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/azurefile-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/azurefile-csi-driver?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fazurefile-csi-driver.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fazurefile-csi-driver?ref=badge_shield)

### About
This driver allows Kubernetes to use [azure file](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) volume, csi plugin name: `file.csi.azure.com`

### Project Status
Status: Beta

### Container Images & CSI Compatibility:
|Azure File CSI Driver Version  | Image                                              | v1.0.0 |
|-------------------------------|----------------------------------------------------|--------|
|master branch                  |mcr.microsoft.com/k8s/csi/azurefile-csi:latest      | yes    |
|v0.3.0                         |mcr.microsoft.com/k8s/csi/azurefile-csi:v0.3.0      | yes    |
|v0.2.0                         |mcr.microsoft.com/k8s/csi/azurefile-csi:v0.2.0      | yes    |

### Kubernetes Compatibility
| Azure File CSI Driver\Kubernetes Version | 1.13+ |
|------------------------------------------|-------|
| master branch                            | yes   |
| v0.3.0                                   | yes   |
| v0.2.0                                   | yes   |

### Driver parameters
Please refer to [`file.csi.azure.com` driver parameters](./docs/driver-parameters.md)
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

### Prerequisite
 - The driver initialization depends on a [Cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md), usually it's `/etc/kubernetes/azure.json` on all kubernetes nodes deployed by AKS or aks-engine, here is an [azure.json example](./deploy/example/azure.json)
 > if cluster is based on Managed Service Identity(MSI), make sure all agent nodes have `Contributor` role for current resource group

### Install azurefile CSI driver on a kubernetes cluster
Please refer to [install azurefile csi driver](https://github.com/kubernetes-sigs/azurefile-csi-driver/blob/master/docs/install-azurefile-csi-driver.md)

### E2E Usage example
#### 1. create a pod with csi azurefile driver mount on linux
##### Option#1: Azurefile Dynamic Provisioning
 - Create an azurefile CSI storage class
```
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/storageclass-azurefile-csi.yaml
```

 - Create an azurefile CSI PVC
```
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi.yaml
```

##### Option#2: Azurefile Static Provisioning(use an existing azure file share)
 - Use `kubectl create secret` to create `azure-secret` with existing storage account name and key
```
kubectl create secret generic azure-secret --from-literal accountname=NAME --from-literal accountkey="KEY" --type=Opaque
```

 - Create an azurefile CSI PV, download `pv-azurefile-csi.yaml` file and edit `shareName` in `volumeAttributes`
```
wget https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/pv-azurefile-csi.yaml
vi pv-azurefile-csi.yaml
kubectl create -f pv-azurefile-csi.yaml
```

 - Create an azurefile CSI PVC which would be bound to the above PV
```
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi-static.yaml
```

#### 2. validate PVC status and create an nginx pod
 - make sure pvc is created and in `Bound` status finally
```
watch kubectl describe pvc pvc-azurefile
```

 - create a pod with azurefile CSI PVC
```
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/nginx-pod-azurefile.yaml
```

#### 3. enter the pod container to do validation
 - watch the status of pod until its Status changed from `Pending` to `Running` and then enter the pod container
```
$ watch kubectl describe po nginx-azurefile
$ kubectl exec -it nginx-azurefile -- bash
root@nginx-azurefile:/# df -h
Filesystem                                                                Size  Used Avail Use% Mounted on
overlay                                                                   30G   19G  11G   65%  /
tmpfs                                                                     3.5G  0    3.5G  0%   /dev
...
//f571xxx.file.core.windows.net/pvc-54caa11f-9e27-11e9-ba7b-0601775d3b69  1.0G  64K  1.0G  1%   /mnt/azurefile
...
```
In the above example, there is a `/mnt/azurefile` directory mounted as cifs filesystem.

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### Links
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/Home.html)
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
