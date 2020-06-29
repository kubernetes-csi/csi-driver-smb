## Integration Test
Integration test verifies the functionality of CSI driver as a standalone server outside Kubernetes. It exercises the lifecycle of the volume by staging, mounting volumes and the reverse operations.

## Run Integration Tests Locally
### Prerequisite
 - make sure `GOPATH` is set and [csc](https://github.com/rexray/gocsi/tree/master/csc) tool is installed under `$GOPATH/bin/csc`
```console
export set GOPATH=/root/go
go get github.com/rexray/gocsi/csc
```

### Run integration tests
```console
export GOBIN="/root/go/bin"
make test-integration
```

