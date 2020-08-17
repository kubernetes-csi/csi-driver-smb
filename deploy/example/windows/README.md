# CSI on Windows example

## Feature Status: Alpha
CSI on Windows support is an alpha feature since Kubernetes v1.18, refer to [Windows-CSI-Support](https://github.com/kubernetes/enhancements/blob/master/keps/sig-windows/20190714-windows-csi-support.md) for more details.

## Prerequisite
- Install CSI-Proxy on Windows Node

[csi-proxy installation](https://github.com/Azure/aks-engine/blob/master/docs/topics/csi-proxy-windows.md) is supported with [aks-engine v0.48.0](https://github.com/Azure/aks-engine/releases/tag/v0.48.0).

## Deploy a Windows pod with PVC mount
### Create a Windows deployment
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/statefulset.yaml
```

### Enter pod container to verify
```
$ kubectl exec -it busybox-smb-0  -- bash
C:/ $ ls mnt/smb
```

In the above example, there is a `c:\mnt\smb` directory mounted as SMB filesystem.
