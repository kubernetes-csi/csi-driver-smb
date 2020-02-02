# azurefile CSI driver for Kubernetes
[![Travis](https://travis-ci.org/kubernetes-sigs/azurefile-csi-driver.svg)](https://travis-ci.org/kubernetes-sigs/azurefile-csi-driver)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/azurefile-csi-driver/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/azurefile-csi-driver?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fazurefile-csi-driver.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fazurefile-csi-driver?ref=badge_shield)

### About
This driver allows Kubernetes to use [azure file](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) volume, csi plugin name: `file.csi.azure.com`

### Container Images & CSI Compatibility:
|Azure File CSI Driver Version  | Image                                              | v1.0.0 |
|-------------------------------|----------------------------------------------------|--------|
|master branch                  |mcr.microsoft.com/k8s/csi/azurefile-csi:latest      | yes    |
|v0.4.0                         |mcr.microsoft.com/k8s/csi/azurefile-csi:v0.4.0      | yes    |
|v0.3.0                         |mcr.microsoft.com/k8s/csi/azurefile-csi:v0.3.0      | yes    |

### Kubernetes Compatibility
| Azure File CSI Driver\Kubernetes Version | 1.14+ |
|------------------------------------------|-------|
| master branch                            | yes   |
| v0.4.0                                   | yes   |
| v0.3.0                                   | yes   |

### Driver parameters
Please refer to [`file.csi.azure.com` driver parameters](./docs/driver-parameters.md)
 > storage class `file.csi.azure.com` parameters are compatible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin

### Prerequisite
 - The driver initialization depends on a [Cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md), usually it's `/etc/kubernetes/azure.json` on all kubernetes nodes deployed by AKS or aks-engine, here is an [azure.json example](./deploy/example/azure.json)
 > if cluster is based on Managed Service Identity(MSI), make sure all agent nodes have `Contributor` role for current resource group

### Install azurefile CSI driver on a kubernetes cluster
Please refer to [install azurefile csi driver](https://github.com/kubernetes-sigs/azurefile-csi-driver/blob/master/docs/install-azurefile-csi-driver.md)

### Examples
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [Snapshot](./deploy/example/snapshot)

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### Links
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/Home.html)
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
