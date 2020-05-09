# Azure File CSI Driver for Kubernetes
[![Travis](https://travis-ci.org/csi-driver/csi-driver-smb.svg)](https://travis-ci.org/csi-driver/csi-driver-smb)
[![Coverage Status](https://coveralls.io/repos/github/csi-driver/csi-driver-smb/badge.svg?branch=master)](https://coveralls.io/github/csi-driver/csi-driver-smb?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fcsi-driver-smb.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fcsi-driver-smb?ref=badge_shield)

### About
This driver allows Kubernetes to use [Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) volume, csi plugin name: `file.csi.azure.com`

### Container Images & Kubernetes Compatibility:
|Azure File CSI Driver Version  | Image                                              | 1.14+  |
|-------------------------------|----------------------------------------------------|--------|
|master branch                  |mcr.microsoft.com/k8s/csi/smb-csi:latest      | yes    |
|v0.6.0                         |mcr.microsoft.com/k8s/csi/smb-csi:v0.6.0      | yes    |
|v0.5.0                         |mcr.microsoft.com/k8s/csi/smb-csi:v0.5.0      | yes    |

### Driver parameters
Please refer to [`file.csi.azure.com` driver parameters](./docs/driver-parameters.md)
 > storage class `file.csi.azure.com` parameters are compatible with built-in [smb](https://kubernetes.io/docs/concepts/storage/volumes/#smb) plugin

### Prerequisite
 - The driver initialization depends on a [Cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md), usually it's `/etc/kubernetes/azure.json` on all kubernetes nodes deployed by [AKS](https://docs.microsoft.com/en-us/azure/aks/) or [aks-engine](https://github.com/Azure/aks-engine), here is an [azure.json example](./deploy/example/azure.json). This driver also supports [read cloud config from kuberenetes secret](./docs/read-from-secret.md).
 > if cluster identity is [Managed Service Identity(MSI)](https://docs.microsoft.com/en-us/azure/aks/use-managed-identity), make sure user assigned identity has `Contributor` role on node resource group

### Install smb CSI driver on a kubernetes cluster
Please refer to [install smb csi driver](https://github.com/csi-driver/csi-driver-smb/blob/master/docs/install-csi-driver-smb.md)

### Examples
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [Snapshot](./deploy/example/snapshot)
 - [VHD disk](./deploy/example/disk)
 - [Windows](./deploy/example/windows)
 
### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md) 

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### Links
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/Home.html)
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
