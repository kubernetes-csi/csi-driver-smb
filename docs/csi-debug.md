## CSI driver debug tips

### Case#1: volume create/delete failed
 - locate csi driver pod
```console
$ kubectl get po -o wide -n kube-system | grep csi-smb-controller
NAME                                     READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-controller-56bfddd689-dh5tk      5/5     Running   0          35s     10.240.0.19    k8s-agentpool-22533604-0
csi-smb-controller-56bfddd689-sl4ll      5/5     Running   0          35s     10.240.0.23    k8s-agentpool-22533604-1
```
 - get csi driver logs
```console
$ kubectl logs csi-smb-controller-56bfddd689-dh5tk -c smb -n kube-system > csi-smb-controller.log
```
> note: there could be multiple controller pods, if there are no helpful logs, try to get logs from other controller pods

### Case#2: volume mount/unmount failed
 - locate csi driver pod and make sure which pod do tha actual volume mount/unmount
```console
$ kubectl get po -o wide -n kube-system | grep csi-smb-node
NAME                                      READY   STATUS    RESTARTS   AGE     IP             NODE
csi-smb-node-cvgbs                        3/3     Running   0          7m4s    10.240.0.35    k8s-agentpool-22533604-1
csi-smb-node-dr4s4                        3/3     Running   0          7m4s    10.240.0.4     k8s-agentpool-22533604-0
```

 - get csi driver logs
```console
$ kubectl logs csi-smb-node-cvgbs -c smb -n kube-system > csi-smb-node.log
```

#### Update driver version quickly by editing driver deployment directly
 - update controller deployment
```console
kubectl edit deployment csi-smb-controller -n kube-system
```
 - update daemonset deployment
```console
kubectl edit ds csi-smb-node -n kube-system
```
change below deployment config, e.g.
```console
        image: mcr.microsoft.com/k8s/csi/smb-csi:v1.1.0
        imagePullPolicy: Always
```

### troubleshooting connection failure on agent node
 - On Linux node
```console
mkdir /tmp/test
sudo mount -v -t cifs //smb-server/fileshare /tmp/test -o vers=3.0,username=accountname,password=accountkey,dir_mode=0777,file_mode=0777,cache=strict,actimeo=30
```

 - Check whether original smb mount directory works
```console
sudo mount | grep cifs
```

 - On Windows node
```console
$User = "AZURE\USERNAME"
$PWord = ConvertTo-SecureString -String "PASSWORD" -AsPlainText -Force
$Credential = New-Object –TypeName System.Management.Automation.PSCredential –ArgumentList $User, $Pword
New-SmbGlobalMapping -LocalPath x: -RemotePath \\smb-server\fileshare -Credential $Credential
Get-SmbGlobalMapping
cd x:
dir
```
