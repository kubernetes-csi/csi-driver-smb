# Installation with Helm 3

Quick start instructions for the setup and configuration of smb CSI driver using Helm.

## Prerequisites

1. [install Helm Client 3.0+ ](https://helm.sh/docs/intro/quickstart/#install-helm)

## Install AzureFile via `helm install`

```console
$ cd $GOPATH/src/github.com/csi-driver/csi-driver-smb/charts/latest
$ helm package csi-driver-smb
$ helm install csi-driver-smb csi-driver-smb-latest.tgz --namespace kube-system
```

## Uninstall

```console
$ helm uninstall csi-driver-smb -n kube-system
```

## The Latest Helm Chart Configuration

The following table lists the configurable parameters of the latest Azure File CSI Driver chart and their default values.

| Parameter                                         | Description                                                | Default                                                           |
|---------------------------------------------------|------------------------------------------------------------|-------------------------------------------------------------------|
| `image.smb.repository`                       | csi-driver-smb docker image                           | mcr.microsoft.com/k8s/csi/smb-csi                            |
| `image.smb.tag`                              | csi-driver-smb docker image tag                       | latest                                                            |
| `image.smb.pullPolicy`                       | csi-driver-smb image pull policy                      | IfNotPresent                                                      |
| `image.csiProvisioner.repository`                 | csi-provisioner docker image                               | mcr.microsoft.com/oss/kubernetes-csi/csi-provisioner              |
| `image.csiProvisioner.tag`                        | csi-provisioner docker image tag                           | v1.4.0                                                            |
| `image.csiProvisioner.pullPolicy`                 | csi-provisioner image pull policy                          | IfNotPresent                                                      |
| `image.csiAttacher.repository`                    | csi-attacher docker image                                  | mcr.microsoft.com/oss/kubernetes-csi/csi-attacher                 |
| `image.csiAttacher.tag`                           | csi-attacher docker image tag                              | v1.2.0                                                            |
| `image.csiAttacher.pullPolicy`                    | csi-attacher image pull policy                             | IfNotPresent                                                      |
| `image.csiSnapshotter.repository`                 | csi-snapshotter docker image                               | mcr.microsoft.com/oss/kubernetes-csi/csi-snapshotter              |
| `image.csiSnapshotter.tag`                        | csi-snapshotter docker image tag                           | v1.1.0                                                            |
| `image.csiSnapshotter.pullPolicy`                 | csi-snapshotter image pull policy                          | IfNotPresent                                                      |
| `image.csiResizer.repository`                     | csi-resizer docker image                                   | mcr.microsoft.com/oss/kubernetes-csi/csi-resizer                  |
| `image.csiResizer.tag`                            | csi-resizer docker image tag                               | v0.3.0                                                            |
| `image.csiResizer.pullPolicy`                     | csi-resizer image pull policy                              | IfNotPresent                                                      |
| `image.livenessProbe.repository`                  | liveness-probe docker image                                | mcr.microsoft.com/oss/kubernetes-csi/livenessprobe                |
| `image.livenessProbe.tag`                         | liveness-probe docker image tag                            | v1.1.0                                                            |
| `image.livenessProbe.pullPolicy`                  | liveness-probe image pull policy                           | IfNotPresent                                                      |
| `image.nodeDriverRegistrar.repository`            | csi-node-driver-registrar docker image                     | mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar    |
| `image.nodeDriverRegistrar.tag`                   | csi-node-driver-registrar docker image tag                 | v1.2.0                                                            |
| `image.nodeDriverRegistrar.pullPolicy`            | csi-node-driver-registrar image pull policy                | IfNotPresent                                                      |
| `serviceAccount.create`                           | whether create service account of csi-smb-controller  | true                                                              |
| `rbac.create`                                     | whether create rbac of csi-smb-controller             | true                                                              |
| `controller.replicas`                             | the replicas of csi-smb-controller                    | 2                                                                 |
| `linux.enabled`                                   | whether enable linux feature                               | true                                                              |
| `windows.enabled`                                 | whether enable windows feature                             | false                                                             |
| `windows.image.livenessProbe.repository`          | windows liveness-probe docker image                        | mcr.microsoft.com/oss/kubernetes-csi/livenessprobe                |
| `windows.image.livenessProbe.tag`                 | windows liveness-probe docker image tag                    | v2.0.1-alpha.1-windows-1809-amd64                                 |
| `windows.image.livenessProbe.pullPolicy`          | windows liveness-probe image pull policy                   | IfNotPresent                                                      |
| `windows.image.nodeDriverRegistrar.repository`    | windows csi-node-driver-registrar docker image             | mcr.microsoft.com/oss/kubernetes-csi/csi-node-driver-registrar    |
| `windows.image.nodeDriverRegistrar.tag`           | windows csi-node-driver-registrar docker image tag         | v1.2.1-alpha.1-windows-1809-amd64                                 |
| `windows.image.nodeDriverRegistrar.pullPolicy`    | windows csi-node-driver-registrar image pull policy        | IfNotPresent                                                      |

## Troubleshooting

If there are some errors when using helm to install, follow the steps to debug:

1. Add `--wait -v=5 --debug` in `helm install` command.
2. Then the error pods  can be located.
3. Use `kubectl describe ` to acquire more info.
4. Check the related resource of the pod, such as serviceaacount, rbac, etc.
