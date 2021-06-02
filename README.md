# SMB CSI Driver for Kubernetes
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-csi/csi-driver-smb/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-csi/csi-driver-smb?branch=master)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fkubernetes-csi%2Fcsi-driver-smb.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkubernetes-csi%2Fcsi-driver-smb?ref=badge_shield)

### About
This driver allows Kubernetes to use [SMB](https://wiki.wireshark.org/SMB) CSI volume, csi plugin name: `smb.csi.k8s.io`

### Project status: GA

### Container Images & Kubernetes Compatibility:
|Driver Version | Image                                    | supported k8s version | supported [Windows csi-proxy](https://github.com/kubernetes-csi/csi-proxy) version |
|---------------|------------------------------------------|-----------------------|-------------------------------------|
|master branch  |mcr.microsoft.com/k8s/csi/smb-csi:latest  | 1.17+                 | v0.2.2+                             |
|v1.0.0         |mcr.microsoft.com/k8s/csi/smb-csi:v1.0.0  | 1.17+                 | v0.2.2+                             |
|v0.6.0         |mcr.microsoft.com/k8s/csi/smb-csi:v0.6.0  | 1.15+                 | v0.2.0+                             |
|v0.5.0         |mcr.microsoft.com/k8s/csi/smb-csi:v0.5.0  | 1.15+                 | v0.2.0+                             |

### Driver parameters
Please refer to [`smb.csi.k8s.io` driver parameters](./docs/driver-parameters.md)

### Install driver on a Kubernetes cluster
 - install by [kubectl](./docs/install-smb-csi-driver.md)
 - install by [helm charts](./charts)
 
### Examples
 - [Set up a Samba Server on a Kubernetes cluster](./deploy/example/smb-provisioner/)
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [Windows](./deploy/example/windows)

### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md) 

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### View CI Results
Check testgrid [sig-storage-csi-smb](https://testgrid.k8s.io/sig-storage-csi-other) dashboard.

### Links
 - [SMB FlexVolume driver](https://github.com/Azure/kubernetes-volume-drivers/tree/master/flexvolume/smb)
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
