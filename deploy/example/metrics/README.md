# Get Prometheus metrics from CSI driver

1. Create `csi-smb-controller` service with targetPort `29644`
```console
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/deploy/example/metrics/csi-smb-controller-svc.yaml
```

2. Get `EXTERNAL-IP` of service `csi-smb-controller`
```console
$ kubectl get svc csi-smb-controller -n kube-system
NAME                 TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)           AGE
csi-smb-controller   LoadBalancer   10.0.217.224   20.39.0.91    29644:32128/TCP   45m
```

3. Run following command to get metrics
```console
ip=`kubectl get svc csi-smb-controller -n kube-system | grep smb | awk '{print $4}'`
curl http://$ip:29644/metrics
```
