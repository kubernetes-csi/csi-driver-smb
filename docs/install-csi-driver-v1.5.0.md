## Install SMB CSI driver v1.5.0 version on a Kubernetes cluster

### Install by kubectl
```console
curl -skSL https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/v1.5.0/deploy/install-driver.sh | bash -s v1.5.0 --
```

 - check pods status:
```console
kubectl -n kube-system get pod -o wide --watch -l app=csi-smb-controller
kubectl -n kube-system get pod -o wide --watch -l app=csi-smb-node
```

example output:

```
NAME                                        READY   STATUS    RESTARTS   AGE     IP            NODE                                NOMINATED NODE   READINESS GATES
csi-smb-controller-788486959d-5qmn7         3/3     Running   0          23s     10.244.0.45   aks-agentpool-60632172-vmss000006   <none>           <none>
csi-smb-controller-788486959d-g4hpm         3/3     Running   0          32s     10.244.1.33   aks-agentpool-60632172-vmss000007   <none>           <none>
csi-smb-node-4gwzl                          3/3     Running   0          15s     10.244.1.34   aks-agentpool-60632172-vmss000007   <none>           <none>
csi-smb-node-hg76w                          3/3     Running   0          27s     10.244.0.44   aks-agentpool-60632172-vmss000006   <none>           <none>
```

### clean up SMB CSI driver
```console
curl -skSL https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/v1.5.0/deploy/uninstall-driver.sh | bash -s --
```
