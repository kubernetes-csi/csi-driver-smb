# Install azurefile CSI driver on a kubernetes cluster

If you have already installed Helm, you can also use it to install azurefile CSI driver. Please see [Installation with Helm](../charts/README.md).

## Installation with kubectl

```
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/crd-csi-driver-registry.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/crd-csi-node-info.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/rbac-csi-attacher.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/rbac-csi-driver-registrar.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/rbac-csi-provisioner.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/rbac-csi-snapshotter.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/csi-azurefile-provisioner.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/csi-azurefile-attacher.yaml
kubectl apply -f https://raw.githubusercontent.com/csi-driver/azurefile-csi-driver/master/deploy/azurefile-csi-driver.yaml
```

- check pods status:

```
watch kubectl get po -o wide -n kube-system | grep csi-azurefile
```

example output:

```
NAME                          READY   STATUS    RESTARTS   AGE   IP            NODE
csi-azurefile-attacher-0      1/1     Running   0          22h   10.240.0.61   k8s-agentpool-17181929-1
csi-azurefile-g2ksx           2/2     Running   0          21h   10.240.0.4    k8s-agentpool-17181929-0
csi-azurefile-nqxn9           2/2     Running   0          21h   10.240.0.35   k8s-agentpool-17181929-1
csi-azurefile-provisioner-0   1/1     Running   0          22h   10.240.0.39   k8s-agentpool-17181929-1
```

- clean up azure file CSI driver

```
kubectl delete ds csi-azurefile -n kube-system
kubectl delete sts csi-azurefile-provisioner -n kube-system
kubectl delete sts csi-azurefile-attacher -n kube-system
```
