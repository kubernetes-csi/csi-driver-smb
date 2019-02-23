# Mock CSI Driver
Extremely simple mock driver used to test `csi-sanity` based on `rexray/gocsi/mock`.
It can be used for testing of Container Orchestrators that implement client side
of CSI interface.

```
Usage of mock:
  -disable-attach
        Disables RPC_PUBLISH_UNPUBLISH_VOLUME capability.
  -name string
        CSI driver name. (default "io.kubernetes.storage.mock")
```

It prints all received CSI messages to stdout encoded as json, so a test can check that
CO sent the right CSI message.

Example of such output:

```
gRPCCall: {"Method":"/csi.v0.Controller/ControllerGetCapabilities","Request":{},"Response":{"capabilities":[{"Type":{"Rpc":{"type":1}}},{"Type":{"Rpc":{"type":3}}},{"Type":{"Rpc":{"type":4}}},{"Type":{"Rpc":{"type":6}}},{"Type":{"Rpc":{"type":5}}},{"Type":{"Rpc":{"type":2}}}]},"Error":""}
gRPCCall: {"Method":"/csi.v0.Controller/ControllerPublishVolume","Request":{"volume_id":"12","node_id":"some-fake-node-id","volume_capability":{"AccessType":{"Mount":{}},"access_mode":{"mode":1}}},"Response":null,"Error":"rpc error: code = NotFound desc = Not matching Node ID some-fake-node-id to Mock Node ID io.kubernetes.storage.mock"}
```
