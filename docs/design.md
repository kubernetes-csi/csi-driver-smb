# azurefile CSI driver design goals
 > azurefile CSI driver is implemented as compatitable as possible with built-in [azurefile](https://kubernetes.io/docs/concepts/storage/volumes/#azurefile) plugin, it has following goals:

Goal | Status | Notes
--- | --- | --- |
Support service principal and msi authentication | Completed |  |
Support both Linux & Windows | In Progress | Windows related work is in progress: [Enable CSI hostpath example on windows](https://github.com/kubernetes-csi/drivers/issues/79) |
Compatible with original storage class parameters and usage| Completed | there is a little difference in static provision, see [example](../README.md#example2-azurefile-static-provisioninguse-an-existing-azure-file-share) |
Support sovereign cloud| Completed |  |
Support volume size grow| to-do |  |
Support snapshot | to-do |  |

### Other work items
work item | Status | Notes
--- | --- | --- |
Complete all unit tests | In Progress |  |
Set up E2E test | to-do |  |
Implement NodeStage/NodeUnstage functions | to-do | two pods on same node could share same mount |
Implement azure file csi driver on Windows | to-do |  |

### Implementation details
To prevent possible regression issues, azurefile CSI driver use [azure cloud provider](https://github.com/kubernetes/kubernetes/tree/v1.13.0/pkg/cloudprovider/providers/azure) library. Thus, all bug fixes in the built-in azure file plugin would be incorporated into this driver.
