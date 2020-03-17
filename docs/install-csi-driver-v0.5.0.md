## Install azurefile CSI driver development version on a Kubernetes cluster

### Install by kubectl

 - option#1

```console
curl -skSL https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/install-driver-standalone.sh | bash -s v0.5.0 --
```

 - option#2

```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/crd-csi-driver-registry.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/crd-csi-node-info.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/rbac-csi-azurefile-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/csi-azurefile-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/csi-azurefile-node.yaml
```

- check pods status:

```console
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

### clean up Azure File CSI driver
 - option#1

```console
curl -skSL https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/uninstall-driver-standalone.sh | bash -s v0.5.0 --
```

 - option#2

```console
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/csi-azurefile-controller.yaml --ignore-not-found
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/csi-azurefile-node.yaml --ignore-not-found
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/crd-csi-driver-registry.yaml --ignore-not-found
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/crd-csi-node-info.yaml --ignore-not-found
kubectl delete -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/v0.5.0/deploy/rbac-csi-azurefile-controller.yaml --ignore-not-found
```
