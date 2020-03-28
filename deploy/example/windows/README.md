# CSI on Windows example

## Feature Status: Alpha

CSI on Windows support is an alpha feature since Kubernetes v1.18, refer to [Windows-CSI-Support](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20190714-windows-csi-support.md) for more details.

## Prerequisite

- Install CSI-Proxy on Windows Node

[csi-proxy installation](https://github.com/Azure/aks-engine/blob/master/docs/topics/csi-proxy-windows.md) is supported with [aks-engine v0.48.0](https://github.com/Azure/aks-engine/releases/tag/v0.48.0).

## Install CSI Driver

Follow the [instructions](https://github.com/kubernetes-sigs/azurefile-csi-driver/blob/master/docs/install-csi-driver-master.md#windows) to install windows version driver.

## Deploy a Windows pod with PVC mount

### Create StorageClass

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/storageclass-azurefile-csi.yaml
```

### Create Windows pod

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/windows/statefulset.yaml
```

### Enter pod container to do validation

```
$ kubectl exec -it aspnet-azurefile-0 -- cmd
Microsoft Windows [Version 10.0.17763.1098]
(c) 2018 Microsoft Corporation. All rights reserved.

C:\inetpub\wwwroot>cd c:\mnt\azurefile

c:\mnt\azurefile>echo hello > 20200328

c:\mnt\azurefile>dir
 Volume in drive C has no label.
 Volume Serial Number is DE36-B78A

 Directory of c:\mnt\azurefile

03/28/2020  05:48 AM    <DIR>          .
03/28/2020  05:48 AM    <DIR>          ..
03/28/2020  05:49 AM                 8 20200328
               1 File(s)              8 bytes
               2 Dir(s)  107,374,116,864 bytes free
```

In the above example, there is a `c:\mnt\azurefile` directory mounted as NTFS filesystem.
