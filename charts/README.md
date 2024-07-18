# Install CSI driver with Helm 3

## Prerequisites

- [install Helm](https://helm.sh/docs/intro/quickstart/#install-helm)

### Tips

- run smb-controller on control plane node: `--set controller.runOnControlPlane=true`
- Microk8s based kubernetes recommended settings:
    - `--set linux.kubelet="/var/snap/microk8s/common/var/lib/kubelet"` - sets correct path to microk8s kubelet even
      though a user has a folder link to it.

### install a specific version

```console
helm repo add csi-driver-smb https://raw.githubusercontent.com/kubernetes-csi/csi-driver-smb/master/charts
helm install csi-driver-smb csi-driver-smb/csi-driver-smb --namespace kube-system --version v1.15.0
```

### install driver with customized driver name, deployment name

> only supported from `v1.2.0`+

- following example would install a driver with name `smb2`

```console
helm install csi-driver-smb2 csi-driver-smb/csi-driver-smb --namespace kube-system --set driver.name="smb2.csi.k8s.io" --set controller.name="csi-smb2-controller" --set rbac.name=smb2 --set serviceAccount.controller=csi-smb2-controller-sa --set serviceAccount.node=csi-smb2-node-sa --set node.name=csi-smb2-node --set node.livenessProbe.healthPort=39643
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

| Parameter                                               | Description                                                                                                | Default                                                 |
|---------------------------------------------------------|------------------------------------------------------------------------------------------------------------|---------------------------------------------------------|
| `driver.name`                                           | alternative driver name                                                                                    | `smb.csi.k8s.io`                                        |
| `feature.enableGetVolumeStats`                          | allow GET_VOLUME_STATS on agent node                                                                       | `false`                                                 |
| `image.baseRepo`                                        | base repository of driver images                                                                           | `registry.k8s.io/sig-storage`                           |
| `image.smb.repository`                                  | csi-driver-smb docker image                                                                                | `gcr.io/k8s-staging-sig-storage/smbplugin`              |
| `image.smb.tag`                                         | csi-driver-smb docker image tag                                                                            | `canary`                                                |
| `image.smb.pullPolicy`                                  | csi-driver-smb image pull policy                                                                           | `IfNotPresent`                                          |
| `image.csiProvisioner.tag`                              | csi-provisioner docker image tag                                                                           | `v5.0.1`                                                |
| `image.csiProvisioner.pullPolicy`                       | csi-provisioner image pull policy                                                                          | `IfNotPresent`                                          |
| `image.livenessProbe.repository`                        | liveness-probe docker image                                                                                | `/livenessprobe`                                        |
| `image.livenessProbe.tag`                               | liveness-probe docker image tag                                                                            | `v2.13.1`                                                |
| `image.livenessProbe.pullPolicy`                        | liveness-probe image pull policy                                                                           | `IfNotPresent`                                          |
| `image.nodeDriverRegistrar.repository`                  | csi-node-driver-registrar docker image                                                                     | `/csi-node-driver-registrar`                            |
| `image.nodeDriverRegistrar.tag`                         | csi-node-driver-registrar docker image tag                                                                 | `v2.11.1`                                                |
| `image.nodeDriverRegistrar.pullPolicy`                  | csi-node-driver-registrar image pull policy                                                                | `IfNotPresent`                                          |
| `imagePullSecrets`                                      | Specify docker-registry secret names as an array                                                           | `[]` (does not add image pull secrets to deployed pods) |
| `serviceAccount.create`                                 | whether create service account of csi-smb-controller                                                       | `true`                                                  |
| `rbac.create`                                           | whether create rbac of csi-smb-controller                                                                  | `true`                                                  |
| `rbac.name`                                             | driver name in rbac role                                                                                   | `true`                                                  |
| `podAnnotations`                                        | collection of annotations to add to all the pods                                                           | `{}`                                                    |
| `podLabels`                                             | collection of labels to add to all the pods                                                                | `{}`                                                    |
| `priorityClassName`                                     | priority class name to be added to pods                                                                    | `system-cluster-critical`                               |
| `securityContext`                                       | security context to be added to pods                                                                       | `{ seccompProfile: {type: RuntimeDefault} }`            |
| `controller.name`                                       | name of driver deployment                                                                                  | `csi-smb-controller`                                    |
| `controller.replicas`                                   | replica num of csi-smb-controller                                                                          | `1`                                                     |
| `controller.dnsPolicy`                                  | dnsPolicy of driver node daemonset, available values: `Default`, `ClusterFirstWithHostNet`, `ClusterFirst` |    `ClusterFirstWithHostNet`                            |
| `controller.metricsPort`                                | metrics port of csi-smb-controller                                                                         | `29644`                                                 |
| `controller.livenessProbe.healthPort `                  | health check port for liveness probe                                                                       | `29642`                                                 |
| `controller.logLevel`                                   | controller driver log level                                                                                | `5`                                                     |
| `controller.workingMountDir`                            | working directory for provisioner to mount smb shares temporarily                                          | `/tmp`                                                  |
| `controller.runOnMaster`                                | run controller on master node                                                                              | `false`                                                 |
| `controller.runOnControlPlane`                          | run controller on control plane node                                                                       | `false`                                                 |
| `controller.resources.csiProvisioner.limits.memory`     | csi-provisioner memory limits                                                                              | `100Mi`                                                 |
| `controller.resources.csiProvisioner.requests.cpu`      | csi-provisioner cpu requests limits                                                                        | `10m`                                                   |
| `controller.resources.csiProvisioner.requests.memory`   | csi-provisioner memory requests limits                                                                     | `20Mi`                                                  |
| `controller.resources.livenessProbe.limits.memory`      | liveness-probe memory limits                                                                               | `300Mi`                                                 |
| `controller.resources.livenessProbe.requests.cpu`       | liveness-probe cpu requests limits                                                                         | `10m`                                                   |
| `controller.resources.livenessProbe.requests.memory`    | liveness-probe memory requests limits                                                                      | `20Mi`                                                  |
| `controller.resources.smb.limits.memory`                | smb-csi-driver memory limits                                                                               | `200Mi`                                                 |
| `controller.resources.smb.requests.cpu`                 | smb-csi-driver cpu requests limits                                                                         | `10m`                                                   |
| `controller.resources.smb.requests.memory`              | smb-csi-driver memory requests limits                                                                      | `20Mi`                                                  |
| `controller.resources.csiResizer.limits.memory`         | csi-resizer memory limits                                                                                  | `300Mi`                                                 |
| `controller.resources.csiResizer.requests.cpu`          | csi-resizer cpu requests limits                                                                            | `10m`                                                   |
| `controller.resources.csiResizer.requests.memory`       | csi-resizer memory requests limits                                                                         | `20Mi`                                                  |
| `controller.affinity`                                   | controller pod affinity                                                                                    | `{}`                                                    |
| `controller.nodeSelector`                               | controller pod node selector                                                                               | `{}`                                                    |
| `controller.tolerations`                                | controller pod tolerations                                                                                 | `[]`                                                    |
| `node.maxUnavailable`                                   | `maxUnavailable` value of csi-smb-node daemonset                                                           | `1`                                                     |
| `node.livenessProbe.healthPort `                        | health check port for liveness probe                                                                       | `29643`                                                 |
| `node.logLevel`                                         | node driver log level                                                                                      | `5`                                                     |
| `node.affinity`                                         | node pod affinity                                                                                          | {}                                                      |
| `node.nodeSelector`                                     | node pod node selector                                                                                     | `{}`                                                    |
| `linux.enabled`                                         | whether enable linux feature                                                                               | `true`                                                  |
| `linux.dsName`                                          | name of driver daemonset on linux                                                                          | `csi-smb-node`                                          |
| `linux.dnsPolicy`                                       | dnsPolicy of driver node daemonset, available values: `Default`, `ClusterFirstWithHostNet`, `ClusterFirst` | `ClusterFirstWithHostNet`                               |
| `linux.kubelet`                                         | configure kubelet directory path on Linux agent node node                                                  | `/var/lib/kubelet`                                      |
| `linux.krb5CacheDirectory`                              | directory for kerberos cache on Linux agent node node, empty string means default                          | `/var/lib/kubelet/kerberos/`                            |
| `linux.krb5Prefix`                                      | prefix for kerberos cache on Linux agent node node, empty string means default                             | `krb5cc_`                                               |
| `linux.resources.livenessProbe.limits.memory`           | liveness-probe memory limits                                                                               | `100Mi`                                                 |
| `linux.resources.livenessProbe.requests.cpu`            | liveness-probe cpu requests limits                                                                         | `10m`                                                   |
| `linux.resources.livenessProbe.requests.memory`         | liveness-probe memory requests limits                                                                      | `20Mi`                                                  |
| `linux.resources.nodeDriverRegistrar.limits.memory`     | csi-node-driver-registrar memory limits                                                                    | `100Mi`                                                 |
| `linux.resources.nodeDriverRegistrar.requests.cpu`      | csi-node-driver-registrar cpu requests limits                                                              | `10m`                                                   |
| `linux.resources.nodeDriverRegistrar.requests.memory`   | csi-node-driver-registrar memory requests limits                                                           | `20Mi`                                                  |
| `linux.resources.smb.limits.memory`                     | smb-csi-driver memory limits                                                                               | `200Mi`                                                 |
| `linux.resources.smb.requests.cpu`                      | smb-csi-driver cpu requests limits                                                                         | `10m`                                                   |
| `linux.resources.smb.requests.memory`                   | smb-csi-driver memory requests limits                                                                      | `20Mi`                                                  |
| `windows.enabled`                                       | whether enable windows feature                                                                             | `false`                                                 |
| `windows.dsName`                                        | name of driver daemonset on windows                                                                        | `csi-smb-node-win`                                      |
| `windows.removeSMBMappingDuringUnmount`                 | remove SMBMapping during unmount on Windows node windows                                                   | `true`                                                  |
| `windows.resources.livenessProbe.limits.memory`         | liveness-probe memory limits                                                                               | `200Mi`                                                 |
| `windows.resources.livenessProbe.requests.cpu`          | liveness-probe cpu requests limits                                                                         | `10m`                                                   |
| `windows.resources.livenessProbe.requests.memory`       | liveness-probe memory requests limits                                                                      | `20Mi`                                                  |
| `windows.resources.nodeDriverRegistrar.limits.memory`   | csi-node-driver-registrar memory limits                                                                    | `200Mi`                                                 |
| `windows.resources.nodeDriverRegistrar.requests.cpu`    | csi-node-driver-registrar cpu requests limits                                                              | `10m`                                                   |
| `windows.resources.nodeDriverRegistrar.requests.memory` | csi-node-driver-registrar memory requests limits                                                           | `20Mi`                                                  |
| `windows.resources.smb.limits.memory`                   | smb-csi-driver memory limits                                                                               | `400Mi`                                                 |
| `windows.resources.smb.requests.cpu`                    | smb-csi-driver cpu requests limits                                                                         | `10m`                                                   |
| `windows.resources.smb.requests.memory`                 | smb-csi-driver memory requests limits                                                                      | `20Mi`                                                  |
| `windows.kubelet`                                       | configure kubelet directory path on Windows agent node                                                     | `'C:\var\lib\kubelet'`                                  |

### Csi Proxy support on windows

The helm can setup the host-process deamonset for the csi proxy, by setting windows.csiproxy.enabled to true.

The following table lists the configurable parameters of the latest CSI-proxy Driver chart and default values.

| Parameter                                               | Description                                                                                                | Default                                                 |
|---------------------------------------------------------|------------------------------------------------------------------------------------------------------------|---------------------------------------------------------|
| `windows.csiproxy.enabled`                              | whether enable csi-proxy daemonset                                                                         | `false`                                                 |
| `windows.csiproxy.dsName`                               | name of driver csi-proxy daemonset on windows                                                              | `csi-proxy-win`                                         |
| `windows.csiproxy.username`                             | name of windows user on the host machine to run the proxy as                                               | `NT AUTHORITY\\SYSTEM`                                  |
| `windows.csiproxy.affinity`                             | controller pod affinity                                                                                    | `{}`                                                    |
| `windows.csiproxy.nodeSelector`                         | controller pod node selector                                                                               | `{"kubernetes.io/os": windows}                          |
| `windows.csiproxy.tolerations`                          | controller pod tolerations                                                                                 | `[]`                                                    |
| `image.csiproxy.repository`                             | csiproxy docker image                                                                                      | `ghcr.io/kubernetes-sigs/sig-windows/csi-proxy`         |
| `image.csiproxy.tag`                                    | csiproxy docker image tag                                                                                  | `v1.1.2`                                                |
| `image.csiproxy.pullPolicy`                             | csiproxy image pull policy                                                                                 | `IfNotPresent`                                          |

## troubleshooting

- Add `--wait -v=5 --debug` in `helm install` command to get detailed error
- Use `kubectl describe` to acquire more info
