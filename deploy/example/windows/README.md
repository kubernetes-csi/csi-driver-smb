# CSI Driver on Windows

## Note
**Only** use root share for one SMB server in one cluster and use `subPath` in deployment, if there is already `\\smb-server\share\test1` mounted, would get error when mounting volume `\\smb-server\share\test2` after Windows node reboot. Workaround is only use `\\smb-server\share` as `source`, details [here](https://github.com/kubernetes-csi/csi-driver-smb/issues/219#issuecomment-781952587).

## Feature Status: GA

## Prerequisite
 > if you have set `windows.useHostProcessContainers` as `true`, csi-proxy is not needed by CSI driver.
- [Install CSI-Proxy on Windows Node](https://github.com/kubernetes-csi/csi-proxy#installation)
- install csi-proxy on k8s 1.23+ Windows node using host process daemonset directly
```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/csi-proxy.yaml
```


## Deploy a Windows pod with PVC mount
### Create a Windows deployment
```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/windows/statefulset.yaml
```

### Enter pod container to verify
```console
$ kubectl exec -it busybox-smb-0  -- bash
C:/ $ ls mnt/smb
```

In the above example, there is a `c:\mnt\smb` directory mounted as SMB filesystem.

## Name resolution

It is important to note that if you have defined your Storage class based on the name of the service, then the Windows NODE must be able to resolve that name (ie. outside Kubernetes).

e.g.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
provisioner: smb.csi.k8s.io
allowVolumeExpansion: true
parameters:
  source: //smb-server.default.svc.cluster.local/share
...
```

and the error you get in logs (`kubectl logs -n kube-system ds/csi-proxy`) is the famous `The network path was not found`:

```
I0706 02:11:47.703698    4460 server.go:56] calling NewSmbGlobalMapping with remote path "\\\\smb-server.default.svc.cluster.local\\share\\pvc-1a6c607b-df88-4a61-8628-4d9d066dc158"
I0706 02:11:48.524376    4460 server.go:98] Remote \\smb-server.default.svc.cluster.local\share not mapped. Mapping now!
E0706 02:11:49.319012    4460 server.go:101] failed NewSmbGlobalMapping NewSmbGlobalMapping failed. output: "New-SmbGlobalMapping : The network path was not found. \r\nAt line:1 char:190\r\n+ ... , $PWord;New-SmbGlobalMapping -RemotePath $Env:smbremotepath -Cred ...\r\n+                 ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\r\n    + CategoryInfo          : NotSpecified: (MSFT_SmbGlobalMapping:ROOT/Microsoft/...mbGlobalMapping) [New-SmbGlobalMa \r\n   pping], CimException\r\n    + FullyQualifiedErrorId : Windows System Error 53,New-SmbGlobalMapping\r\n \r\n", err: exit status 1
```

For most cloud providers that offer Windows nodes, this is not an issue. Those nodes will be part of a domain where name resolution includes the Kubernetes dns resolution.

For self-hosted Kubernetes clusters there is no default name resolution to solve this issue.

### Name resolution solution

A simple _hack_ workaround is to use a fixed ClusterIP for the smb service, and use the IP in the storageClass.

It is VERY VERY much not recommended to use fixed IPs inside Kubernetes, but in this very specific case it make for a simple workaround.

The IP needs to be in the service CIDR range, e.g. if it is `10.96.0.0/12`, you could pick `10.111.254.254`.

```yaml
apiVersion: v1
kind: Service
metadata:
...
spec:
  type: ClusterIP
  clusterIP: 10.111.254.254
...
```
and
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: smb
provisioner: smb.csi.k8s.io
allowVolumeExpansion: true
parameters:
  source: //10.111.254.254/share
...
```

If `kubectl logs -n kube-system ds/csi-proxy` continues to use names after this fix, it might be necessary to reboot the node.

