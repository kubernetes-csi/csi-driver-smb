## End to End Test

## Run E2E tests Locally
### Prerequisite
- Set up a Kubernetes cluster (with version >= 1.13) using [aks-engine](https://github.com/Azure/aks-engine) or [AKS](https://docs.microsoft.com/en-us/azure/aks/)
- `$KUBECONFIG` is set or your kubeconfig is under `$HOME/.kube/config`
- Export the following environment varibles:
```bash
export TENANT_ID=<your tenant ID>
export SUBSCRIPTION_ID=<the Azure subscription ID that your cluster is under>
export AAD_CLIENT_ID=<the service principal ID that your cluster is using>
export AAD_CLIENT_SECRET=<the service principal password that your cluster is using>
export RESOURCE_GROUP=<the resource group that your cluster is under>
export LOCATION=<the location of your resource group>
```

To run the E2E tests:

```bash
docker login # Login to docker so the test image can be pushed to your Docker Hub
REGISTRY=<your Docker Hub ID / registry> make e2e-test
# Run E2E tests on an existing Azure File CSI Driver image
REGISTRY=<your Docker Hub ID / registry> IMAGE_VERSION=<desired image version> make e2e-test
```
