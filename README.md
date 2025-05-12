# SMB CSI Driver for Kubernetes
![linux build status](https://github.com/kubernetes-csi/csi-driver-smb/actions/workflows/linux.yaml/badge.svg)
![windows build status](https://github.com/kubernetes-csi/csi-driver-smb/actions/workflows/windows.yaml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/kubernetes-csi/csi-driver-smb/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-csi/csi-driver-smb?branch=master)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/csi-driver-smb)](https://artifacthub.io/packages/search?repo=csi-driver-smb)

### About
This driver allows Kubernetes to access [SMB](https://wiki.wireshark.org/SMB) server on both Linux and Windows nodes, plugin name: `smb.csi.k8s.io`. The driver requires existing configured SMB server, it supports dynamic provisioning of Persistent Volumes via Persistent Volume Claims by creating a new sub directory under SMB server.

### Project status: GA

### Container Images & Kubernetes Compatibility:
|Driver Version | supported k8s version | supported [Windows csi-proxy](https://github.com/kubernetes-csi/csi-proxy) version |
|---------------|-----------------------|-------------------------------------|
|master branch  | 1.21+                 | v0.2.2+                             |
|v1.18.0        | 1.21+                 | v0.2.2+                             |
|v1.17.0        | 1.21+                 | v0.2.2+                             |
|v1.16.0        | 1.21+                 | v0.2.2+                             |

### Driver parameters
Please refer to `smb.csi.k8s.io` [driver parameters](./docs/driver-parameters.md)

### Install driver on a Kubernetes cluster
 - install via [helm charts](./charts)
 - install via [kubectl](./docs/install-smb-csi-driver.md)
 
### Examples
 - [How to Use the Windows CSI Proxy and CSI SMB Driver for Kubernetes](https://www.phillipsj.net/posts/how-to-use-the-windows-csi-proxy-and-csi-smb-driver-for-kubernetes/)
 - [Set up a Samba Server on a Kubernetes cluster](./deploy/example/smb-provisioner/)
 - [Basic usage](./deploy/example/e2e_usage.md)
 - [Windows](./deploy/example/windows)
 - [Volume cloning](./deploy/example/cloning)

### Troubleshooting
 - [CSI driver troubleshooting guide](./docs/csi-debug.md) 

## Kubernetes Development
Please refer to [development guide](./docs/csi-dev.md)

### View CI Results
 - testgrid [sig-storage-csi-smb](https://testgrid.k8s.io/sig-storage-csi-other) dashboard.
 - Driver image build pipeline: [post-csi-driver-smb-push-images](https://testgrid.k8s.io/sig-storage-image-build#post-csi-driver-smb-push-images)

### Links
 - [SMB FlexVolume driver](https://github.com/Azure/kubernetes-volume-drivers/tree/master/flexvolume/smb)
 - [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
