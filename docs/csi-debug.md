## CSI driver debug tips
### Case#1: volume mount/unmount failed
 - locate csi driver pod and make sure which pod do tha actual volume mount/unmount
```console
$ kubectl get po -o wide -n kube-system | grep csi-smb-node
NAME                                            READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-smb-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

 - get csi driver logs
```console
$ kubectl logs csi-smb-node-cvgbs -c smb -n kube-system > csi-smb-node.log
```
