# SMB CSI Driver for Kubernetes
[![Travis](https://travis-ci.org/csi-driver/csi-driver-smb.svg)](https://travis-ci.org/csi-driver/csi-driver-smb)
[![Coverage Status](https://coveralls.io/repos/github/csi-driver/csi-driver-smb/badge.svg?branch=master)](https://coveralls.io/github/csi-driver/csi-driver-smb?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fcsi-driver-smb.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-sigs%2Fcsi-driver-smb?ref=badge_shield)

### About
This driver allows Kubernetes to use [SMB](https://wiki.wireshark.org/SMB) CSI volume, csi plugin name: `smb.csi.k8s.io`

### Container Images & Kubernetes Compatibility:
|SMB CSI Driver Version  | Image                                | 1.14+  |
|-------------------------------|-------------------------------|--------|
|master branch                  |andyzhangx/smb-csi:latest      | yes    |

### Driver parameters
Please refer to [`smb.csi.k8s.io` driver parameters](./docs/driver-parameters.md)

### Install smb CSI driver on a kubernetes cluster
Please refer to [install smb csi driver](./docs/install-csi-driver-master.md)

### Examples
 - [Basic usage](./deploy/example/e2e_usage.md)
 
### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md) 

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### Links
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/Home.html)
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
