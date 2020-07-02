## Set up a SMB server deployment on a Kubernetes cluster
This page will show you how to set up a SMB server deployment on a Kubernetes cluster, the file share data is stored on local disk.

 - Use `kubectl create secret` to create `smbcreds` with SMB username, password
```console
kubectl create secret generic smbcreds --from-literal username=USERNAME --from-literal password="PASSWORD"
```

 - Create a SMB server deployment
> modify `/smbshare-volume` in deployment to specify another path to store smb share data
```console
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/smb-server.yaml
```

After deployment, a new service `smb-server` will be together with a load balancer(has one Public IP Address)

 - Get public ip address of service `smb-server`
```console
# kubectl get service smb-server --watch
NAME         TYPE           CLUSTER-IP   EXTERNAL-IP    PORT(S)         AGE
smb-server   LoadBalancer   10.0.23.79   20.43.192.64   445:30612/TCP   89s
```

In above example, the new SMB file share is `//20.43.192.64/share`

- Test SMB share mount on local machine
```console
mount -t cifs //20.43.192.64/share local-directory -o vers=3.0,username=username,password=test,dir_mode=0777,file_mode=0777,cache=strict,actimeo=30
```

- [CSI driver basic usage on SMB file share](../e2e_usage.md)