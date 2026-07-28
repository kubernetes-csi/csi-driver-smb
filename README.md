# csi-driver-smb helm chart repository

This branch hosts the Helm chart repository for csi-driver-smb via
GitHub Pages. Published automatically by the
`Publish Helm Chart to GitHub Pages` workflow.

To use:

    helm repo add csi-driver-smb https://kubernetes-csi.github.io/csi-driver-smb
    helm repo update csi-driver-smb
    helm install csi-driver-smb csi-driver-smb/csi-driver-smb \
      --namespace kube-system --version <x.y.z>
