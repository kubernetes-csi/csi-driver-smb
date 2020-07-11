## Set up a Samba Server and a deployment to access Samba Server on a Kubernetes cluster
This page will show you how to:
 - set up a Samba Server deployment on a Kubernetes cluster
 > file share data is stored on local disk.
 - set up a deployment to access Samba Server on a Kubernetes cluster

### Set up a Samba Server
 - Use `kubectl create secret` to create `smbcreds` with SMB username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```

 - Create a Samba Server deployment
> modify `/smbshare-volume` in deployment to specify different path for smb share data store
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/smb-server.yaml
```

After deployment, a new service `smb-server` is created, file share path is `//smb-server.default.svc.cluster.local/share`

### Create a deployment to access above Samba Server
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/pv-smb.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/pvc-smb-static.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/deployment.yaml
```

 - Verification
```console
# kubectl exec -it  deployment-smb-646c5d579c-5sc6n bash
root@deployment-smb-646c5d579c-5sc6n:/# df -h
Filesystem                                    Size  Used Avail Use% Mounted on
...
//smb-server.default.svc.cluster.local/share   97G   21G   76G  22% /mnt/smb
/dev/sda1                                      97G   21G   76G  22% /etc/hosts
...
```
