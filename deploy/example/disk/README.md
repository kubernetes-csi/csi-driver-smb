## Azure File CSI driver fast attach disk feature example
Attach Azure disks in < 1 second. Attach as many as you want. VHD disk(based on azure file) feature could mount Azure disks as Linux block device directly on VMs without dependency on the host.

 - Motivation: [Metadata/namespace heavy workload on Azure File](https://docs.microsoft.com/en-us/azure/storage/files/storage-troubleshooting-files-performance#cause-2-metadatanamespace-heavy-workload)
 
Add a VHD on the file share and mount VHD over SMB from the client to perform files operations against the data. This approach works for single writer and multiple readers scenarios and allows metadata operations to be local, offering performance similar to a local direct-attached storage. 

 - performance

Scheduling 20 pods with one vhd disk each on **one** node **in parallel** could be completed in 2min, while for azure managed disk driver, it's 30min.

#### Feature Status
Status: Alpha

#### 1. create a pod with vhd disk mount on Linux
##### Option#1: Dynamic Provisioning
 - Create an azurefile CSI storage class and PVC
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/disk/storageclass-azurefile-csi.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/disk/pvc-azurefile-disk.yaml
```

##### Option#2: Static Provisioning(use an existing vhd file in azure file share)
> make sure credential in cluster could access that file share
 - Create an azurefile CSI storage class and PVC
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/disk/storageclass-azurefile-existing-disk.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/disk/pvc-azurefile-disk.yaml
```

#### 2. validate PVC status and create an nginx pod
 - make sure pvc is created and in `Bound` status finally
```console
watch kubectl describe pvc pvc-azurefile
```

 - create a pod with azurefile CSI PVC
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/azurefile-csi-driver/master/deploy/example/nginx-pod-azurefile.yaml
```

#### 3. enter the pod container to do validation
 - watch the status of pod until its Status changed from `Pending` to `Running` and then enter the pod container
```console
$ kubect exec -it nginx-azurefile bash
# df -h
Filesystem      Size  Used Avail Use% Mounted on
...
/dev/loop0       98G   61M   98G   1% /mnt/azurefile
/dev/sda1        29G   16G   14G  53% /etc/hosts
...
```
In the above example, there is a `/mnt/azurefile` directory mounted as ext4 filesystem.
