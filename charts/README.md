# Installation with Helm

Quick start instructions for the setup and configuration of azurefile CSI driver using Helm.

## Prerequisites

1. [install Helm Client](https://helm.sh/docs/using_helm/#installing-the-helm-client)

2. [initialize Helm and install Tiller](https://helm.sh/docs/using_helm/#initialize-helm-and-install-tiller)

## Install AzureFile via `helm install`

```console
$ cd $GOPATH/src/sigs.k8s.io/azurefile-csi-driver/charts/latest
$ helm package azurefile-csi-driver
$ helm install azurefile-csi-driver azurefile-csi-driver-latest.tgz --namespace kube-system
```

## Uninstall

```console
$ helm delete --purge azurefile-csi-driver
```
