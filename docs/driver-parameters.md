## `smb.csi.k8s.io` driver parameters
This driver only supports static provisioning

 - Static Provision(use existing smb)
  > get a quick example [here](../deploy/example/pv-smb-csi.yaml)

Name | Meaning | Available Value | Mandatory | Default value
--- | --- | --- | --- | ---
volumeAttributes.source | SMB server address | `//IP/smb-server/directory`(for azure file, format is `//accountname.file.core.windows.net/filesharename`) | Yes |
nodeStageSecretRef.name | secret name that stores `username`, `password` | existing secret name |  Yes  |
nodeStageSecretRef.namespace | namespace where the secret is | k8s namespace  |  No  | `default`
