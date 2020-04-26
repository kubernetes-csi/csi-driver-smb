## CSI driver debug tips
### Case#1: volume create/delete issue
 - locate csi driver pod
```console
$ kubectl get po -o wide -n kube-system | grep csi-azurefile-controller
NAME                                            READY   STATUS    RESTARTS   AGE     IP             NODE
csi-azurefile-controller-56bfddd689-dh5tk       5/5     Running   0          35s     10.240.0.19    k8s-agentpool-22533604-0
csi-azurefile-controller-56bfddd689-sl4ll       5/5     Running   0          35s     10.240.0.23    k8s-agentpool-22533604-1
```
 - get csi driver logs
```console
$ kubectl logs `kubectl get po -n kube-system | grep csi-azurefile-controller | cut -d ' ' -f1` -c azurefile -n kube-system > csi-azurefile-controller.log
```
> note: there could be multiple controller pods, if there are no helpful logs, try to get logs from other controller pods

### Case#2: volume mount/unmount failed
 - locate csi driver pod and make sure which pod do tha actual volume mount/unmount
```console
$ kubectl get po -o wide -n kube-system | grep csi-azurefile-node
NAME                                            READY   STATUS    RESTARTS   AGE     IP             NODE
csi-azurefile-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-azurefile-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

 - get csi driver logs
```console
$ kubectl logs csi-azurefile-node-cvgbs -c azurefile -n kube-system > csi-azurefile-node.log
```
