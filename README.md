# SMB CSI Driver for Kubernetes
[![Travis](https://travis-ci.org/kubernetes-csi/csi-driver-smb.svg)](https://travis-ci.org/kubernetes-csi/csi-driver-smb)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-csi/csi-driver-smb/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-csi/csi-driver-smb?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-csi%2Fcsi-driver-smb.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-csi%2Fcsi-driver-smb?ref=badge_shield)

### About
This driver allows Kubernetes to use [SMB](https://wiki.wireshark.org/SMB) CSI volume, csi plugin name: `smb.csi.k8s.io`

### Project status: alpha

### Container Images & Kubernetes Compatibility:
|SMB CSI Driver Version  | Image                                        | 1.14+  |
|------------------------|----------------------------------------------|--------|
|master branch           |mcr.microsoft.com/k8s/csi/smb-csi:latest      | yes    |
|v0.2.0                  |mcr.microsoft.com/k8s/csi/smb-csi:v0.2.0      | yes    |
|v0.1.0                  |mcr.microsoft.com/k8s/csi/smb-csi:v0.1.0      | yes    |

### Driver parameters
Please refer to [`smb.csi.k8s.io` driver parameters](./docs/driver-parameters.md)

### Install SMB CSI driver on a kubernetes cluster
Please refer to [install SMB CSI driver](./docs/install-smb-csi-driver.md)

### Examples
 - [Set up a Samba Server on a Kubernetes cluster](./deploy/example/smb-provisioner/)
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [Windows](./deploy/example/windows)

### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md) 

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### Links
 - [SMB FlexVolume driver](https://github.com/Azure/kubernetes-volume-drivers/tree/master/flexvolume/smb)
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
