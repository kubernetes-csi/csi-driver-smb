# Sanity Test Command Line Program
This is the command line program that tests a CSI driver using the [`sanity`](https://github.com/kubernetes-csi/csi-test/tree/master/pkg/sanity) package test suite.

Example:

```
$ csi-sanity --csi.endpoint=<your csi driver endpoint>
```

If you want to specify a mount point:

```
$ csi-sanity --csi.endpoint=<your csi driver endpoint> --csi.mountpoint=/mnt
```

For verbose type:

```
$ csi-sanity --ginkgo.v --csi.endpoint=<your csi driver endpoint>
```

For csi-credentials, create a secrets file with all the secrets in it:
```yaml
CreateVolumeSecret:
  secretKey: secretval1
DeleteVolumeSecret:
  secretKey: secretval2
ControllerPublishVolumeSecret:
  secretKey: secretval3
ControllerUnpublishVolumeSecret:
  secretKey: secretval4
NodeStageVolumeSecret:
  secretKey: secretval5
NodePublishVolumeSecret:
  secretKey: secretval6
```

Pass the file path to csi-sanity as:
```
$ csi-sanity --csi.endpoint=<your csi driver endpoint> --csi.secrets=<path to secrets file>
```

Replace the keys and values of the credentials appropriately. Since the whole
secret is passed in the request, multiple key-val pairs can be used.

### Help
The full Ginkgo and golang unit test parameters are available. Type

```
$ csi-sanity -h
```

to get more information

### Download

Please see the [Releases](https://github.com/kubernetes-csi/csi-test/releases) page
to download the latest version of `csi-sanity`
