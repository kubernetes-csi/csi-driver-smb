## Run a samba server kubernetes deployment 

### create a deployment and service

```
kubectl create secret generic smb-server-creds --from-literal username=username --from-literal password="test"
kubectl create -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/smb-provisioner/smb-server-deployment.yaml
```

### mount shared directory 
```
mount -t cifs //smb-server-ip/share local-directory -o vers=3.0,username=username,password=test,domain=userdomain,dir_mode=0777,file_mode=0777,cache=strict,actimeo=30
```  
 - Note:  
Username and password should be the same as those set in secret smb-server-creds    
Other configuration could refer to [depson/samba](https://github.com/dperson/samba#configuration)
