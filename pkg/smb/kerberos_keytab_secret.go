/*
Copyright 2026 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smb

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// The secret of a volume is only passed to NodeStageVolume, so a keytab
// rotated in the secret (after a password change of the principal) would
// only be used when a volume of the principal is staged again on the node.
// With --krb5-update-keytab-from-secret, when the renewal with the stored
// keytab of a principal fails, the driver reads the secrets of the
// principal's volumes (the nodeStageSecretRef of their PersistentVolume) and
// uses a new keytab found there when kinit succeeds with it. A keytab that
// does not work never replaces the stored one.

// keytabSecretTimeout bounds the API calls of one renewal.
const keytabSecretTimeout = time.Minute

// keytabCandidate is a keytab of the secret of a volume that differs from
// the stored keytab of the volume's credential uid.
type keytabCandidate struct {
	source string // namespace/name of the secret
	keytab []byte
}

// keytabSecretCandidates returns, per credential uid of principals, the
// keytabs in the secrets of its volumes that differ from its stored keytab
// and are for the same principal.
func (d *Driver) keytabSecretCandidates(principals map[int]string) map[int][]keytabCandidate {
	candidates := map[int][]keytabCandidate{}
	if !d.krb5UpdateKeytabFromSecret || d.kubeClient == nil || len(principals) == 0 {
		return candidates
	}
	ctx, cancel := context.WithTimeout(context.Background(), keytabSecretTimeout)
	defer cancel()

	d.krb5Mutex.Lock()
	files, _ := filepath.Glob(filepath.Join(d.keytabDirectory(), "*"+krb5VolumeFileSuffix))
	var volumes []keytabVolume
	for _, file := range files {
		if volume, ok := readKeytabVolume(file); ok && volume.VolumeID != "" && principals[volume.CredUID] != "" {
			volumes = append(volumes, volume)
		}
	}
	d.krb5Mutex.Unlock()

	d.resolveKeytabSecretRefs(ctx, volumes)

	type secretKeytab struct {
		principal string
		keytab    []byte
		ok        bool
	}
	secrets := map[string]secretKeytab{}
	stored := map[int][]byte{}
	for _, volume := range volumes {
		if volume.SecretName == "" {
			continue
		}
		source := volume.SecretNamespace + "/" + volume.SecretName
		secret, seen := secrets[source]
		if !seen {
			secret = secretKeytab{}
			s, err := d.kubeClient.CoreV1().Secrets(volume.SecretNamespace).Get(ctx, volume.SecretName, metav1.GetOptions{})
			if err != nil {
				klog.Warningf("failed to get secret %s of kerberos volume(%s): %v", source, volume.VolumeID, err)
			} else {
				data := map[string]string{}
				for k, v := range s.Data {
					data[k] = string(v)
				}
				principal, keytab, found, err := getKerberosKeytab(data)
				if err != nil {
					klog.Warningf("secret %s of kerberos volume(%s): %v", source, volume.VolumeID, err)
				} else if found {
					secret = secretKeytab{principal: principal, keytab: keytab, ok: true}
				}
			}
			secrets[source] = secret
		}
		principal := principals[volume.CredUID]
		if !secret.ok {
			continue
		}
		if secret.principal != principal {
			// the volume keeps its principal until it is staged again
			klog.V(4).Infof("secret %s of volume(%s) is for %s, the volume is mounted as %s: not used", source, volume.VolumeID, secret.principal, principal)
			continue
		}
		if _, ok := stored[volume.CredUID]; !ok {
			stored[volume.CredUID], _ = os.ReadFile(d.keytabFile(volume.CredUID, krb5KeytabFileSuffix))
		}
		if bytes.Equal(secret.keytab, stored[volume.CredUID]) {
			continue
		}
		duplicate := false
		for _, candidate := range candidates[volume.CredUID] {
			if bytes.Equal(candidate.keytab, secret.keytab) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			candidates[volume.CredUID] = append(candidates[volume.CredUID], keytabCandidate{source: source, keytab: secret.keytab})
		}
	}
	return candidates
}

// resolveKeytabSecretRefs finds the secret of the volumes that do not know it
// yet, from the nodeStageSecretRef of their PersistentVolume, and records it
// in their volume file. The secret of a PersistentVolume can not change, so
// this is done once per volume.
func (d *Driver) resolveKeytabSecretRefs(ctx context.Context, volumes []keytabVolume) {
	missing := map[string]int{}
	for i, volume := range volumes {
		if volume.SecretName == "" {
			missing[volume.VolumeID] = i
		}
	}
	if len(missing) == 0 {
		return
	}
	pvs, err := d.kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Warningf("failed to list the persistent volumes to find the secrets of kerberos volumes: %v", err)
		return
	}
	for _, pv := range pvs.Items {
		csiSource := pv.Spec.CSI
		if csiSource == nil || csiSource.Driver != d.Name || csiSource.NodeStageSecretRef == nil {
			continue
		}
		i, ok := missing[csiSource.VolumeHandle]
		if !ok {
			continue
		}
		volumes[i].SecretNamespace = csiSource.NodeStageSecretRef.Namespace
		volumes[i].SecretName = csiSource.NodeStageSecretRef.Name
		if err := d.recordKeytabSecretRef(volumes[i]); err != nil {
			klog.Warningf("failed to record the secret of kerberos volume(%s): %v", volumes[i].VolumeID, err)
		}
	}
}

// recordKeytabSecretRef stores the secret of a volume in its volume file, if
// the volume is still staged with the same credential uid.
func (d *Driver) recordKeytabSecretRef(volume keytabVolume) error {
	d.krb5Mutex.Lock()
	defer d.krb5Mutex.Unlock()
	file := d.keytabVolumeFile(volume.VolumeID)
	current, ok := readKeytabVolume(file)
	if !ok || current.CredUID != volume.CredUID || current.VolumeID != volume.VolumeID {
		return nil
	}
	current.SecretNamespace, current.SecretName = volume.SecretNamespace, volume.SecretName
	return writeKeytabVolumeFile(file, current)
}

// updateKeytabFromSecrets tries the keytabs of the secrets of the volumes of
// credUID that differ from its stored keytab. The first one with which kinit
// succeeds replaces the stored keytab and gives the new ticket. It returns
// whether the ticket was renewed, and errKeytabReleased when the last volume
// of the principal was unstaged meanwhile.
func (d *Driver) updateKeytabFromSecrets(credUID int, principal string, candidates []keytabCandidate) (bool, error) {
	for _, candidate := range candidates {
		err := d.refreshKeytabTicket(credUID, principal, candidate.keytab)
		if err == nil {
			klog.Infof("updated the kerberos keytab of %s (cruid %d) from secret %s", principal, credUID, candidate.source)
			krb5KeytabUpdates.WithLabelValues(krb5RenewalResultSuccess).Inc()
			return true, nil
		}
		if errors.Is(err, errKeytabReleased) {
			return false, err
		}
		klog.Warningf("the keytab in secret %s does not work for %s (cruid %d), keeping the stored keytab: %v", candidate.source, principal, credUID, err)
		krb5KeytabUpdates.WithLabelValues(krb5RenewalResultFailure).Inc()
	}
	return false, nil
}
