## CSI driver debug tips

### Case#1: volume create/delete failed
 - locate csi driver pod
```console
$ kubectl get po -o wide -n kube-system | grep csi-smb-controller
NAME                                     READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-controller-56bfddd689-dh5tk      5/5     Running   0          35s     10.240.0.19    k8s-agentpool-22533604-0
csi-smb-controller-56bfddd689-sl4ll      5/5     Running   0          35s     10.240.0.23    k8s-agentpool-22533604-1
```
 - get csi driver logs
```console
$ kubectl logs csi-smb-controller-56bfddd689-dh5tk -c smb -n kube-system > csi-smb-controller.log
```
> note: there could be multiple controller pods, if there are no helpful logs, try to get logs from other controller pods

### Case#2: volume mount/unmount failed
 - locate csi driver pod and make sure which pod do tha actual volume mount/unmount
```console
$ kubectl get po -o wide -n kube-system | grep csi-smb-node
NAME                                      READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-smb-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

 - get csi driver logs
```console
$ kubectl logs csi-smb-node-cvgbs -c smb -n kube-system > csi-smb-node.log
```
