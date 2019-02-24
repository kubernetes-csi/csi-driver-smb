## Sanity Tests
Testing the Azure File CSI driver using the [`sanity`](https://github.com/kubernetes-csi/csi-test/tree/master/pkg/sanity) package test suite.

## Run Integration Tests Locally
### Prerequisite
 - make sure `GOPATH` is set

 - set the environment variable AZURE_CREDENTIAL_FILE with the path to cloud provider config file only if you have the file at a different location than `/etc/kubernetes/azure.json`
 > By default Cloud provider config file is present at `/etc/kubernetes/azure.json` on a kubernetes cluster node
```
export set AZURE_CREDENTIAL_FILE=
```

### Run integration tests
```
make test-sanity
```

