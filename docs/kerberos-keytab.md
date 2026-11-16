## Mount with a Kerberos keytab (Linux)

A volume can be mounted with Kerberos in two ways:

 - with a ticket cache in the secret, see [Kerberos ticket support for Linux](./driver-parameters.md#kerberos-ticket-support-for-linux). The ticket in the secret expires, usually after 10 hours. Later mounts of the volume then fail until the secret is updated.
 - with a keytab in the secret, described here. The driver gets the ticket itself with `kinit` when the volume is mounted on a node, and renews it for as long as the volume is mounted. The secret only has to be updated when the keytab changes, see [Rotating the keytab](#rotating-the-keytab).

### Prerequisites
 - The conditions of [Kerberos ticket support for Linux](./driver-parameters.md#these-are-the-conditions-that-must-be-met) apply: `cifs-utils` on every node, the kerberos cache directory, and nodes that look for the ticket caches in that directory (`linux.krb5AutoConfig=true` in the Helm chart).
 - The smb container must be able to find the KDC, see [Finding the KDC](#finding-the-kdc).

### Create the secret
The secret takes two keys:
 - `krb5keytab`: the keytab, Base64-encoded
 - `krb5principal`: the principal to authenticate as

```console
kubectl create secret generic smbcreds-krb5 \
  --from-literal krb5keytab=$(base64 -w 0 user.keytab) \
  --from-literal krb5principal=USERNAME@EXAMPLE.COM
```

### Mount flags
Mount flags should include **sec=krb5** (or **sec=krb5i**). Leave out `cruid=`: the driver adds it.

> See example of the [PersistentVolume](../deploy/example/pv-smb-krb5-keytab.yaml)

There is only one ticket cache per credential uid (`cruid`) on a node. Volumes mounted as different principals therefore need different credential uids, also when they are used by the same pod. The driver gives every principal a credential uid of its own (2000000000 or higher), shared by all volumes mounted as that principal on the node.

If the mount flags do have `cruid=`, that credential uid is used. Mounting a volume with a `cruid` that is in use on the node by another principal, or by volumes mounted with a ticket cache, is refused.

### Ticket renewal
A ticket is only used when a session with the SMB server is set up. The kernel sets up a new session when the connection to the server was lost, e.g. after a network failure or a restart of the server, and needs a valid ticket for that. The driver therefore gets a fresh ticket for every principal with mounted volumes every `--krb5-renew-interval` (`linux.krb5RenewInterval` in the Helm chart, 1 hour by default, `0` disables it), and when it starts.

A renewal that fails is logged (`failed to renew the kerberos ticket of ...`), and the tickets that were obtained before stay in use until they expire. With `--metrics-address` (`node.metricsPort` in the Helm chart) the node plugin exports:
 - `csi_smb_kerberos_ticket_renewal_failing_principals`: the number of principals whose ticket could not be renewed at the last renewal. Alert when it is above 0, e.g. after a password change, see [Rotating the keytab](#rotating-the-keytab).
 - `csi_smb_kerberos_ticket_renewals_total{result="success|failure"}`: the number of renewals.
 - `csi_smb_kerberos_keytab_principals`: the number of principals with volumes mounted with a keytab on the node.

### What is stored on the node
The driver keeps the following in the kerberos cache directory of the node (`/var/lib/kubelet/kerberos/` by default), readable by root only:
 - `keytabs/<cruid>.keytab` and `keytabs/<cruid>.principal`: the keytab and the principal
 - `keytabs/<hash>.volume`: one file per volume mounted with the keytab, with its `cruid`, volume id and staging path
 - `krb5cc_<cruid>`: the ticket cache, owned by the `cruid` user

They are removed when the last volume mounted as the principal is unmounted from the node. A keytab that does not work (no ticket can be obtained with it) is not stored, and does not replace the keytab of volumes that are already mounted.

When a mount fails after its `NodeStageVolume` call timed out, and the volume is not mounted again (e.g. the pod was deleted), the volume is never unmounted from the node. The driver releases the keytab of such volumes when it starts and before every renewal: the volumes whose staging path is no longer mounted.

### Changing the secret of a mounted volume
A volume that is mounted keeps the principal and the `cruid` it was mounted with: the kernel uses that `cruid` until the volume is mounted again on the node. When the principal in the secret, or the `cruid` mount flag, changes while the volume is mounted, the driver keeps the old ones (and logs a warning) until the volume is unmounted from the node.

A new keytab for the same principal replaces the stored keytab when a volume of the principal is mounted on the node, also when kubelet stages a volume that is already mounted again (e.g. after a restart of kubelet), and only when a ticket can be obtained with it. When no ticket can be obtained for a volume that is already mounted, e.g. because the KDC can not be reached, the volume keeps its ticket cache: the driver logs a warning and the renewal tries again later.

### Rotating the keytab
A keytab stops working when the password of the principal changes, e.g. when the password of an Active Directory account is changed or expires: `kinit` then fails with `Preauthentication failed`. Update the secret with a keytab made after the change.

The driver only gets the secret when a volume is staged on a node. A node therefore keeps using the old keytab until a volume of the principal is staged there again:
 - a volume of the principal that is mounted on the node for the first time,
 - a volume that is unmounted from the node, because no pod on the node uses it any more, and mounted again by the next pod,
 - kubelet staging the mounted volumes again, e.g. after a restart of kubelet.

Starting one more pod with a volume that is already mounted on the node does not stage it again. The tickets obtained with the old keytab stay valid until they expire (usually after 10 hours); after that the kernel can not set up a new session when the connection to the server is lost, and the volume becomes unavailable. After a password change, make sure that the volumes of the principal are staged again on every node before then, e.g. by restarting all the pods on a node that use them.

`kinit` and `cifs.upcall` run in the smb container and must find the KDC of the realm of the principal. There are two ways:

 - **Through DNS.** This needs no configuration when the DNS domain has SRV records for the realm (`_kerberos._udp.<realm>` and `_kerberos._tcp.<realm>`) and the smb container can resolve them. An Active Directory domain always has them. You can check with `dig +short SRV _kerberos._tcp.example.com`.
 - **With `[realms]` settings**, for a KDC that is not in DNS or when a specific KDC must be used:

```
[realms]
  EXAMPLE.COM = {
    kdc = kdc1.example.com
    kdc = kdc2.example.com
  }
```

Either way, a `[domain_realm]` section is needed when the realm of the SMB server can not be derived from its host name (the upper-cased DNS domain of the server is used by default), and setting `default_realm` is good practice:

```
[libdefaults]
  default_realm = EXAMPLE.COM

[domain_realm]
  .example.com = EXAMPLE.COM
  example.com = EXAMPLE.COM
```

Pass these settings with `linux.krb5Conf` of the Helm chart (`--set-file linux.krb5Conf=krb5-realm.conf`). They are mounted in the smb container as `/etc/krb5.conf.d/custom.conf`, and with `linux.krb5AutoConfig=true` also written on every node as `/etc/krb5.conf.d/91-csi-driver-smb-custom.conf`. When the driver is not installed with the Helm chart, mount a file with these settings in `/etc/krb5.conf.d/` of the smb container.

Principals of several realms can be used when the KDCs of each realm can be found in one of these ways. To mount a share of a server in another realm than the principal, the realms need to trust each other.

### OpenShift
Install the Helm chart with:
```console
helm install csi-driver-smb csi-driver-smb/csi-driver-smb --namespace kube-system \
  --set rbac.openshiftPrivilegedSCC=true \
  --set linux.krb5CacheDirectory=/var/lib/kubelet/kerberos/ \
  --set linux.krb5AutoConfig=true \
  --set-file linux.krb5Conf=krb5-realm.conf
```
 - `rbac.openshiftPrivilegedSCC` lets the pods of the driver use the `privileged` SecurityContextConstraints.
 - `linux.krb5Conf` is optional, see [Finding the KDC](#finding-the-kdc).
 - The credential uids have no passwd entry on the node. The driver adds one in the smb container (`linux.krb5EnsurePasswdEntry`, enabled by default), which `cifs.upcall` needs.

### Troubleshooting
 - `failed to get a kerberos ticket for ... with the keytab` in the events of the pod: `kinit` failed, the message has its output. `Cannot contact any KDC` or `Cannot find KDC for realm`: see [Finding the KDC](#finding-the-kdc). `Preauthentication failed`: the keytab does not match the password of the principal (it was changed after the keytab was made, see [Rotating the keytab](#rotating-the-keytab)), or the keytab was made with another salt or encryption type than the KDC expects.
 - `mount error(126): Required key not available`: `cifs.upcall` could not get a ticket for the SMB server. It logs the reason in the system log of the node (`journalctl -t cifs.upcall`). Use the host name of the server in `source`, not its IP address or an alias without a `cifs/` service principal.
 - `mount error(13): Permission denied`: Kerberos worked, the server refuses the principal access to the share or the directory.
 - To see the tickets of a credential uid, run `klist -c /var/lib/kubelet/kerberos/krb5cc_<cruid>` in the smb container of the `csi-smb-node` pod on the node.
