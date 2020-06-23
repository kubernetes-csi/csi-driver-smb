# CSI on Windows example

## Feature Status: Alpha
CSI on Windows support is an alpha feature since Kubernetes v1.18, refer to [Windows-CSI-Support](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20190714-windows-csi-support.md) for more details.

## Prerequisite
- Install CSI-Proxy on Windows Node

[csi-proxy installation](https://github.com/Azure/aks-engine/blob/master/docs/topics/csi-proxy-windows.md) is supported with [aks-engine v0.48.0](https://github.com/Azure/aks-engine/releases/tag/v0.48.0).

## Deploy a Windows pod with PVC mount
### Create a Windows deployment
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/deployment.yaml
```

### Enter pod container to verify
```
$ kubectl exec -it aspnet-smb-0 -- cmd
Microsoft Windows [Version 10.0.17763.1098]
(c) 2018 Microsoft Corporation. All rights reserved.

C:\inetpub\wwwroot>cd c:\mnt\smb

c:\mnt\smb>echo hello > 20200328

c:\mnt\smb>dir
 Volume in drive C has no label.
 Volume Serial Number is DE36-B78A

 Directory of c:\mnt\smb

03/28/2020  05:48 AM    <DIR>          .
03/28/2020  05:48 AM    <DIR>          ..
03/28/2020  05:49 AM                 8 20200328
               1 File(s)              8 bytes
               2 Dir(s)  107,374,116,864 bytes free
```

In the above example, there is a `c:\mnt\smb` directory mounted as NTFS filesystem.
