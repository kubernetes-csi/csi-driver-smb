## End to End Test

## Run E2E tests Locally
### Prerequisite
- Set up a Kubernetes cluster (with version >= 1.13) using [aks-engine](https://github.com/Azure/aks-engine) or [AKS](https://docs.microsoft.com/en-us/azure/aks/)
- `$KUBECONFIG` is set or your kubeconfig is under `$HOME/.kube/config`
- Export the following environment varibles:
```bash
export tenantId=<your tenant ID>
export subscriptionId=<the Azure subscription ID that your cluster is under>
export aadClientId=<the service principal ID that your cluster is using>
export aadClientSecret=<the service principal password that your cluster is using>
export resourceGroup=<the resource group that your cluster is under>
export location=<the location of your resource group>
```

To run the E2E tests:

```bash
docker login # Login to docker so the test image can be pushed to your Docker Hub
REGISTRY=<your Docker Hub ID> make e2e-test
```
