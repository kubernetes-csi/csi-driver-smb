# smb CSI driver design goals
 > smb CSI driver is implemented as compatitable as possible with built-in [smb](https://kubernetes.io/docs/concepts/storage/volumes/#smb) plugin, it has following goals:

Goal | Status | Notes
--- | --- | --- |
Support Kubernetes release 1.12 or later | Completed| release prior to 1.12 won't be supported |
Support service principal and msi authentication | Completed |  |
Support both Linux & Windows | In Progress | Windows related work is in progress: [Enable CSI hostpath example on windows](https://github.com/kubernetes-csi/drivers/issues/79) |
Compatible with original storage class parameters and usage| Completed | there is a little difference in static provision, see [example](../deploy/example/pv-smb-csi.yaml) |
Support sovereign cloud| done | verification pass on Azure China |

### Work items
Item | Status | Notes
--- | --- | --- |
Support volume size grow | to-do |  |
Support snapshot | to-do |  |
Enable CI on Windows | done |  |
Complete all unit tests | Completed |  |
Set up E2E test | to-do |  |
Implement NodeStage/NodeUnstage functions | to-do | two pods on same node could share same mount |
Implement smb csi driver on Windows | to-do |  |

### Implementation details
To prevent possible regression issues, smb CSI driver use [azure cloud provider](https://github.com/kubernetes/kubernetes/tree/v1.13.0/pkg/cloudprovider/providers/azure) library. Thus, all bug fixes in the built-in smb plugin would be incorporated into this driver.
