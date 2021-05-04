# Install CSI driver with Helm 3

## Prerequisites
 - [install Helm](https://helm.sh/docs/intro/quickstart/#install-helm)

## install latest version
```console
helm repo add csi-driver-smb https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts
helm install csi-driver-smb csi-driver-smb/csi-driver-smb --namespace kube-system
```

### install a specific version
```console
helm repo add csi-driver-smb https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts
helm install csi-driver-smb csi-driver-smb/csi-driver-smb --namespace kube-system --version v0.6.0
```

### search for all available chart versions
```console
helm search repo -l csi-driver-smb
```

## uninstall CSI driver
```console
helm uninstall csi-driver-smb -n kube-system
```

## latest chart configuration

The following table lists the configurable parameters of the latest SMB CSI Driver chart and default values.

| Parameter                                         | Description                                                | Default                                                           |
|---------------------------------------------------|------------------------------------------------------------|-------------------------------------------------------------------|
| `image.smb.repository`                            | csi-driver-smb docker image                                | mcr.microsoft.com/k8s/csi/smb-csi                                 |
| `image.smb.tag`                                   | csi-driver-smb docker image tag                            | latest                                                            |
| `image.smb.pullPolicy`                            | csi-driver-smb image pull policy                           | IfNotPresent                                                      |
| `image.csiProvisioner.repository`                 | csi-provisioner docker image                               | mcr.microsoft.com/oss/kubernetes-csi/csi-provisioner              |
| `image.csiProvisioner.tag`                        | csi-provisioner docker image tag                           | v1.4.0                                                            |
| `image.csiProvisioner.pullPolicy`                 | csi-provisioner image pull policy                          | IfNotPresent                                                      |
| `image.livenessProbe.repository`                  | liveness-probe docker image                                | mcr.microsoft.com/oss/kubernetes-csi/livenessprobe                |
| `image.livenessProbe.tag`                         | liveness-probe docker image tag                            | v1.1.0                                                            |
| `image.livenessProbe.pullPolicy`                  | liveness-probe image pull policy                           | IfNotPresent                                                      |
| `image.nodeDriverRegistrar.repository`            | csi-node-driver-registrar docker image                     | mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar    |
| `image.nodeDriverRegistrar.tag`                   | csi-node-driver-registrar docker image tag                 | v1.2.0                                                            |
| `image.nodeDriverRegistrar.pullPolicy`            | csi-node-driver-registrar image pull policy                | IfNotPresent                                                      |
| `imagePullSecrets`                                | Specify docker-registry secret names as an array           | [] (does not add image pull secrets to deployed pods)             |
| `serviceAccount.create`                           | whether create service account of csi-smb-controller       | true                                                              |
| `rbac.create`                                     | whether create rbac of csi-smb-controller                  | true                                                              |
| `controller.replicas`                             | the replicas of csi-smb-controller                                  | 2                                                   |
| `controller.metricsPort`                          | metrics port of csi-smb-controller                   |29644                                               |
| `controller.logLevel`                             | controller driver log level                                                          |`5`                                                           |
| `node.metricsPort`                                | metrics port of csi-smb-node                         |29645
| `node.logLevel`                                   | node driver log level                                                          |`5`                                                           |
| `linux.enabled`                                   | whether enable linux feature                               | true                                                              |
| `windows.enabled`                                 | whether enable windows feature                             | false                                                             |
| `windows.image.livenessProbe.repository`          | windows liveness-probe docker image                        | mcr.microsoft.com/oss/kubernetes-csi/livenessprobe                |
| `windows.image.livenessProbe.tag`                 | windows liveness-probe docker image tag                    | v2.0.1-alpha.1-windows-1809-amd64                                 |
| `windows.image.livenessProbe.pullPolicy`          | windows liveness-probe image pull policy                   | IfNotPresent                                                      |
| `windows.image.nodeDriverRegistrar.repository`    | windows csi-node-driver-registrar docker image             | mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar    |
| `windows.image.nodeDriverRegistrar.tag`           | windows csi-node-driver-registrar docker image tag         | v1.2.1-alpha.1-windows-1809-amd64                                 |
| `windows.image.nodeDriverRegistrar.pullPolicy`    | windows csi-node-driver-registrar image pull policy        | IfNotPresent                                                      |
| `kubelet.linuxPath`                               | configure the kubelet path for Linux node                  | `/var/lib/kubelet`                                                |
| `kubelet.windowsPath`                             | configure the kubelet path for Windows node                | `'C:\var\lib\kubelet'`                                            |
| `controller.runOnMaster`                          | run controller on master node                              | false                                                             |
| `node.livenessProbe.healthPort `                  | the health check port for liveness probe                   | `29643` |

## troubleshooting
 - Add `--wait -v=5 --debug` in `helm install` command to get detailed error
 - Use `kubectl describe` to acquire more info
