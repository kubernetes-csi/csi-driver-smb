## Integration Test
Integration test verifies the functionality of CSI driver as a standalone server outside Kubernetes. It exercises the lifecycle of the volume by creating, attaching, staging, mounting volumes and the reverse operations.

## Run Integration Tests Locally
### Prerequisite
 - make sure `GOPATH` is set and [csc](https://github.com/rexray/gocsi/tree/master/csc) tool is installed under `$GOPATH/bin/csc`
```
export set GOPATH=/root/go
go get github.com/rexray/gocsi/csc
```

 - set Azure credentials by environment variables
 > you could get these variables from `/etc/kubernetes/azure.json` on a kubernetes cluster node
```
export set tenantId=
export set subscriptionId=
export set aadClientId=
export set aadClientSecret=
export set resourceGroup=
export set location=
```

### Run integration tests
```
make test-integration
```
