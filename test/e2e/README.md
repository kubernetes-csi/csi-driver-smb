## End to End Test

## Run E2E tests Locally
### Prerequisite
 - Make sure a kubernetes cluster(with version >= 1.13) is set up and kubeconfig is under `$HOME/.kube/config`
 
### How to run E2E tests

```bash
# Using CSI Driver
make e2e-test

# Run in a Windows cluster
export TEST_WINDOWS="true"
make e2e-test
```
