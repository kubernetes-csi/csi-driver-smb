# Read cloud config from Kubernetes secrets

- Available driver version: `v0.7.0` or above

This driver also supports [reading the cloud config from Kubernetes secrets](https://github.com/kubernetes-sigs/cloud-provider-azure/blob/master/docs/cloud-provider-config.md#setting-azure-cloud-provider-from-kubernetes-secrets). The secret is a serialized version of `azure.json` file with key cloud-config. The secret should be put in `kube-system` namespace and its name should be `azure-cloud-provider`.

### How to convert cloud config to a Kubernetes secret 
1.  create `azure.json` file and fill in all necessary fields, refer to [Cloud provider config](https://github.com/kubernetes-sigs/cloud-provider-azure/blob/master/docs/cloud-provider-config.md), and here is an [example](https://github.com/andyzhangx/demo/blob/master/aks-engine/deployment/etc/kubernetes/azure.json)

2. serialize `azure.json` by following command:
```console
cat azure.json | base64 | awk '{printf $0}'; echo
```

3. create a secret file(`azure-cloud-provider.yaml`) with following values and fill in `cloud-config` value produced in step#2
```yaml
apiVersion: v1
data:
  cloud-config: [fill-in-here]
kind: Secret
metadata:
  name: azure-cloud-provider
  namespace: kube-system
type: Opaque
```

4. Create a `azure-cloud-provider` secret in k8s cluster
```console
kubectl create -f azure-cloud-provider.yaml
```
