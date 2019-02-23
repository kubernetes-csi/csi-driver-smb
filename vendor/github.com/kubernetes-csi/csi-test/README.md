[![Build Status](https://travis-ci.org/kubernetes-csi/csi-test.svg?branch=master)](https://travis-ci.org/kubernetes-csi/csi-test)
[![Docker Repository on Quay](https://quay.io/repository/k8scsi/mock-driver/status "Docker Repository on
Quay")](https://quay.io/repository/k8scsi/mock-driver)

# csi-test
csi-test houses packages and libraries to help test CSI client and plugins.

## For Container Orchestration Tests
CO developers can use this framework to create drivers based on the
[Golang mock](https://github.com/golang/mock) framework. Please see
[co_test.go](test/co_test.go) for an example.

### Mock driver for testing
We also provide a container called `quay.io/k8scsi/mock-driver:canary` which can be used as an in-memory mock driver.
It follows the same release cycle as other containers, so the latest release is `quay.io/k8scsi/mock-driver:v0.3.0`.

You will need to setup the environment variable `CSI_ENDPOINT` for the mock driver to know where to create the unix
domain socket.

## For CSI Driver Tests
To test drivers please take a look at [pkg/sanity](https://github.com/kubernetes-csi/csi-test/tree/master/pkg/sanity).
This package and [csi-sanity](https://github.com/kubernetes-csi/csi-test/tree/master/cmd/csi-sanity) are meant to test
the CSI API capability of a driver. They are meant to be an additional test to the unit, functional, and e2e tests of a
CSI driver.

### Note

* Master is for CSI v0.4.0. Please see the branches for other CSI releases.
* Only Golang 1.9+ supported. See [gRPC issue](https://github.com/grpc/grpc-go/issues/711#issuecomment-326626790)

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack channel](https://kubernetes.slack.com/messages/sig-storage)
- [Mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-storage)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
