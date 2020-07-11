## Set up a Samba Server on a Kubernetes cluster
This page will show you how to set up a Samba Server deployment on a Kubernetes cluster.
 > file share data is stored on local disk.

 - Use `kubectl create secret` to create `smbcreds` secret storing Samba Server username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```

 - Option#1. Create a Samba Server deployment on local disk
> modify `/smbshare-volume` in deployment to specify different path for smb share data store
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/smb-server.yaml
```

 - Option#2. Create a Samba Server deployment on Azure data disk
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/smb-server-azuredisk.yaml
```

After deployment, a new service `smb-server` is created, file share path is `//smb-server.default.svc.cluster.local/share`
