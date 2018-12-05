# azurefile CSI driver for Kubernetes (Alpha)

**WARNING**: This driver is in ALPHA currently. Do NOT use this driver in a production environment in its current state.

 - supported Kubernetes version: v1.12.0 or later version
 - supported agent OS: Linux

> Note: This driver only works before v1.12.0 since there is a CSI breaking change in v1.12.0, find details [here](https://github.com/Azure/kubernetes-volume-drivers/issues/8)

# About
This driver allows Kubernetes to use [azure file](https://docs.microsoft.com/en-us/azure/storage/files/storage-files-introduction) volume

# Prerequisite
 - To ensure that all necessary features are enabled, set the following feature gate flags to true:
```
--feature-gates=CSIPersistentVolume=true,MountPropagation=true,VolumeSnapshotDataSource=true,KubeletPluginsWatcher=true,CSINodeInfo=true,CSIDriverRegistry=true
```
CSIPersistentVolume is enabled by default in v1.10. MountPropagation is enabled by default in v1.10. VolumeSnapshotDataSource is a new alpha feature in v1.12. KubeletPluginsWatcher is enabled by default in v1.12. CSINodeInfo and CSIDriverRegistry are new alpha features in v1.12.

 - An [Cloud provider config file](https://github.com/kubernetes/cloud-provider-azure/blob/master/docs/cloud-provider-config.md) should already exist on all agent nodes
 > usually it's `/etc/kubernetes/azure.json` deployed by AKS or acs-engine, and currently it only supports service principal

# Install azurefile CSI driver on a kubernetes cluster
```
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/crd-csi-driver-registry.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/crd-csi-node-info.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/rbac-csi-attacher.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/rbac-csi-driver-registrar.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/rbac-csi-provisioner.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/rbac-csi-snapshotter.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/csi-azurefile-provisioner.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/csi-azurefile-attacher.yaml
kubectl apply -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/azurefile-csi-driver.yaml
```

 - check pods status:
```
watch kubectl get po -o wide
```
example output:
```
NAME                          READY   STATUS    RESTARTS   AGE   IP            NODE                     
csi-azurefile-attacher-0      1/1     Running   0          22h   10.240.0.61   k8s-agentpool-17181929-1
csi-azurefile-g2ksx           2/2     Running   0          21h   10.240.0.4    k8s-agentpool-17181929-0
csi-azurefile-nqxn9           2/2     Running   0          21h   10.240.0.35   k8s-agentpool-17181929-1
csi-azurefile-provisioner-0   1/1     Running   0          22h   10.240.0.39   k8s-agentpool-17181929-1
```

# Basic Usage
## 1. create a pod with csi azurefile driver mount on linux
#### Example#1: Azurefile Dynamic Provisioning
 - Create a azurefile CSI storage class
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/csi-azurefile-sc.yaml
```

 - Create a azurefile CSI PVC volume
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/pvc-azurefile-csi.yaml
```
make sure pvc is created successfully
```
watch kubectl describe pvc pvc-azurefile
```

 - create a pod with azurefile CSI PVC
```
kubectl create -f https://raw.githubusercontent.com/andyzhangx/azurefile-csi-driver/master/deploy/example/nginx-pod-azurefile.yaml
```

## 3. enter the pod container to do validation
 - watch the status of pod until its Status changed from `Pending` to `Running`
```
watch kubectl describe po nginx-azurefile
```
 - enter the pod container
```
kubectl exec -it nginx-azurefile -- bash
root@nginx-azurefile:/# df -h
Filesystem                                                                                             Size  Used Avail Use% Mounted on
overlay                                                                                                 30G   19G   11G  65% /
tmpfs                                                                                                  3.5G     0  3.5G   0% /dev
tmpfs                                                                                                  3.5G     0  3.5G   0% /sys/fs/cgroup
/dev/sda1                                                                                               30G   19G   11G  65% /etc/hosts
//f5713de20cde511e8ba4900.file.core.windows.net/pvc-file-dynamic-e2ade9f3-f88b-11e8-8429-000d3a03e7d7  1.0G   64K  1.0G   1% /mnt/azurefile
shm                                                                                                     64M     0   64M   0% /dev/shm
tmpfs                                                                                                  3.5G   12K  3.5G   1% /run/secrets/kubernetes.io/serviceaccount
tmpfs                                                                                                  3.5G     0  3.5G   0% /sys/firmware
```
In the above example, there is a `/mnt/azurefile` directory mounted as dysk filesystem.

### Links
 - [Analysis of the CSI Spec](https://blog.thecodeteam.com/2017/11/03/analysis-csi-spec/)
 - [CSI Drivers](https://github.com/kubernetes-csi/drivers)
 - [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)
