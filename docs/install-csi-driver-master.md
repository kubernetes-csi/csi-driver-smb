## Install smb CSI driver development version on a Kubernetes cluster

### Install by kubectl
```console
curl -skSL https://raw.githubusercontent.com/csi-driver/csi-driver-smb/master/deploy/install-driver.sh | bash -s master --
```

 - check pods status:
```console
kubectl -n kube-system get pod -o wide --watch -l app=csi-smb-controller
kubectl -n kube-system get pod -o wide --watch -l app=csi-smb-node
```

example output:

```
NAME                                      READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-smb-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

### clean up SMB CSI driver
```console
curl -skSL https://raw.githubusercontent.com/csi-driver/csi-driver-smb/master/deploy/uninstall-driver.sh | bash -s --
```