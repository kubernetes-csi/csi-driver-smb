## Install SMB CSI driver v1.18.0 version on a Kubernetes cluster
If you have already installed Helm, you can also use it to install this driver. Please check [Installation with Helm](../charts/README.md).

### Install by kubectl
 - Option#1. remote install
```console
curl -skSL https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/v1.18.0/deploy/install-driver.sh | bash -s v1.18.0 --
```

 - Option#2. local install
```console
git clone https://github.com/kubernetes-csi/csi-driver-smb.git
cd csi-driver-smb
git checkout v1.18.0
./deploy/install-driver.sh v1.18.0 local
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
csi-smb-node-4gwzl                          3/3     Running   0          15s     10.244.1.34   aks-agentpool-60632172-vmss000007   <none>           <none>
csi-smb-node-hg76w                          3/3     Running   0          27s     10.244.0.44   aks-agentpool-60632172-vmss000006   <none>           <none>
```

### clean up SMB CSI driver
 - Option#1. remote uninstall
```console
curl -skSL https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/v1.18.0/deploy/uninstall-driver.sh | bash -s --
```

 - Option#2. local uninstall
```console
git clone https://github.com/kubernetes-csi/csi-driver-smb.git
cd csi-driver-smb
git checkout v1.18.0
./deploy/uninstall-driver.sh v1.18.0 local
```
