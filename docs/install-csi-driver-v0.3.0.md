# Install azurefile CSI driveri v0.3.0 on a kubernetes cluster

## Installation with kubectl

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/crd-csi-driver-registry.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/crd-csi-node-info.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/rbac-csi-azurefile-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/csi-azurefile-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/csi-azurefile-node.yaml
```

- check pods status:

```
kubectl -n kube-system get pod -o wide --watch -l app=csi-azurefile-controller
kubectl -n kube-system get pod -o wide --watch -l app=csi-azurefile-node
```

example output:

```
NAME                                            READY   STATUS    RESTARTS   AGE     IP             NODE
csi-azurefile-controller-56bfddd689-dh5tk       6/6     Running   0          35s     10.240.0.19    k8s-agentpool-22533604-0
csi-azurefile-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-azurefile-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

- clean up azure file CSI driver

```
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/csi-azurefile-controller.yaml
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/csi-azurefile-node.yaml
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/crd-csi-driver-registry.yaml
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/crd-csi-node-info.yaml
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/v0.3.0/rbac-csi-azurefile-controller.yaml
```
