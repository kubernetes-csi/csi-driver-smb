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
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/component-base/metrics/testutil"
)

// fakeKinitRejecting is like fakeKinit, but fails for keytabs whose content
// starts with "bad" or is in revoked, as kinit does for a keytab made before
// a password change.
func fakeKinitRejecting(t *testing.T, revoked map[string]bool) *int {
	t.Helper()
	calls := 0
	origKinit, origChown := kinitFunc, chownFunc
	t.Cleanup(func() { kinitFunc, chownFunc = origKinit, origChown })
	chownFunc = func(string, int, int) error { return nil }
	kinitFunc = func(_ context.Context, keytabPath, principal, ccachePath string) error {
		calls++
		keytab, err := os.ReadFile(keytabPath)
		if err != nil {
			return err
		}
		if strings.HasPrefix(string(keytab), "bad") || revoked[string(keytab)] {
			return fmt.Errorf("kinit: Preauthentication failed")
		}
		return os.WriteFile(ccachePath, []byte(fmt.Sprintf("%s %s #%d", principal, keytab, calls)), 0600)
	}
	return &calls
}

func keytabSecret(namespace, name, principal, keytab string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Data: map[string][]byte{
			krb5PrincipalSecretKey: []byte(principal),
			krb5KeytabSecretKey:    []byte(base64.StdEncoding.EncodeToString([]byte(keytab))),
		},
	}
}

func smbPV(name, volumeHandle, secretNamespace, secretName string) *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.PersistentVolumeSpec{PersistentVolumeSource: v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{
			Driver:             DefaultDriverName,
			VolumeHandle:       volumeHandle,
			NodeStageSecretRef: &v1.SecretReference{Namespace: secretNamespace, Name: secretName},
		}}},
	}
}

func counterValue(t *testing.T, result string) float64 {
	t.Helper()
	v, err := testutil.GetCounterMetricValue(krb5KeytabUpdates.WithLabelValues(result))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func updateSecret(t *testing.T, client *fake.Clientset, secret *v1.Secret) {
	t.Helper()
	if _, err := client.CoreV1().Secrets(secret.Namespace).Update(context.Background(), secret, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestRenewKerberosTicketsUpdatesKeytabFromSecret(t *testing.T) {
	d := newKeytabTestDriver(t)
	revoked := map[string]bool{}
	fakeKinitRejecting(t, revoked)
	d.Name = DefaultDriverName
	d.krb5UpdateKeytabFromSecret = true
	client := fake.NewSimpleClientset(
		smbPV("pv1", "vol1", "app", "smb-creds"),
		keytabSecret("app", "smb-creds", testPrincipal, "keytab1"),
	)
	d.kubeClient = client

	flags, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab1"))
	if err != nil {
		t.Fatal(err)
	}
	credUID := credUIDOfFlags(t, flags)
	storedKeytab := func() string { return readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)) }
	successes, failures := counterValue(t, krb5RenewalResultSuccess), counterValue(t, krb5RenewalResultFailure)

	// the stored keytab works: renewed with it, no API calls
	if renewed := d.renewKerberosTickets(); renewed != 1 {
		t.Fatalf("renewed = %d, want 1", renewed)
	}
	if n := len(client.Actions()); n != 0 {
		t.Errorf("API calls = %d, want 0 while the stored keytab works", n)
	}

	// another keytab that works in the secret is not used while the stored
	// one works
	updateSecret(t, client, keytabSecret("app", "smb-creds", testPrincipal, "keytab-alt"))
	d.renewKerberosTickets()
	if got := storedKeytab(); got != "keytab1" {
		t.Errorf("stored keytab = %q, want keytab1", got)
	}

	// the password changes: the stored keytab fails, the new keytab of the
	// secret replaces it and gives the ticket; the secret of the volume is
	// found through its PersistentVolume and recorded
	revoked["keytab1"] = true
	updateSecret(t, client, keytabSecret("app", "smb-creds", testPrincipal, "keytab2"))
	if renewed := d.renewKerberosTickets(); renewed != 1 {
		t.Fatalf("renewed = %d, want 1", renewed)
	}
	if got := storedKeytab(); got != "keytab2" {
		t.Errorf("stored keytab = %q, want keytab2", got)
	}
	if got := readFile(t, d.kerberosCacheFile(credUID)); !strings.HasPrefix(got, testPrincipal+" keytab2 ") {
		t.Errorf("ticket cache = %q, want a ticket from keytab2", got)
	}
	if got := counterValue(t, krb5RenewalResultSuccess) - successes; got != 1 {
		t.Errorf("successful keytab updates = %v, want 1", got)
	}
	if v, _ := readKeytabVolume(d.keytabVolumeFile("vol1")); v.SecretNamespace != "app" || v.SecretName != "smb-creds" {
		t.Errorf("volume file = %+v, want the secret app/smb-creds", v)
	}
	if gauge, _ := testutil.GetGaugeMetricValue(krb5FailingRenewals); gauge != 0 {
		t.Errorf("failing principals = %v, want 0", gauge)
	}

	// the new keytab works: no API calls any more
	calls := len(client.Actions())
	d.renewKerberosTickets()
	if n := len(client.Actions()); n != calls {
		t.Errorf("API calls = %d, want none while the stored keytab works", n-calls)
	}

	// the stored keytab fails and the secret has none that works: the
	// renewal fails and the stored keytab is kept
	revoked["keytab2"] = true
	updateSecret(t, client, keytabSecret("app", "smb-creds", testPrincipal, "bad keytab3"))
	if renewed := d.renewKerberosTickets(); renewed != 0 {
		t.Fatalf("renewed = %d, want 0", renewed)
	}
	if got := storedKeytab(); got != "keytab2" {
		t.Errorf("stored keytab = %q, want keytab2", got)
	}
	if got := counterValue(t, krb5RenewalResultFailure) - failures; got != 1 {
		t.Errorf("failed keytab updates = %v, want 1", got)
	}
	if gauge, _ := testutil.GetGaugeMetricValue(krb5FailingRenewals); gauge != 1 {
		t.Errorf("failing principals = %v, want 1", gauge)
	}

	// a secret for another principal is not used: the volume keeps its
	// principal until it is staged again
	updateSecret(t, client, keytabSecret("app", "smb-creds", testOtherPrincipal, "keytab4"))
	if renewed := d.renewKerberosTickets(); renewed != 0 {
		t.Errorf("renewed = %d, want 0", renewed)
	}
	if got := storedKeytab(); got != "keytab2" {
		t.Errorf("stored keytab = %q, want keytab2", got)
	}

	// the PersistentVolumes were listed once, while the secret was unknown
	lists := 0
	for _, action := range client.Actions() {
		if action.GetVerb() == "list" && action.GetResource().Resource == "persistentvolumes" {
			lists++
		}
	}
	if lists != 1 {
		t.Errorf("lists of persistent volumes = %d, want 1", lists)
	}

	// staging the mounted volume again keeps the recorded secret
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab2")); err != nil {
		t.Fatal(err)
	}
	if v, _ := readKeytabVolume(d.keytabVolumeFile("vol1")); v.SecretName != "smb-creds" {
		t.Errorf("volume file after staging again = %+v, want the secret kept", v)
	}
}

func TestRenewKerberosTicketsKeytabFromSecretsOfSeveralVolumes(t *testing.T) {
	d := newKeytabTestDriver(t)
	fakeKinitRejecting(t, map[string]bool{"keytab": true})
	d.Name = DefaultDriverName
	d.krb5UpdateKeytabFromSecret = true
	// one principal used in two namespaces; only one secret has the new
	// keytab, and a PersistentVolume of another driver has the same handle
	d.kubeClient = fake.NewSimpleClientset(
		smbPV("pv1", "vol1", "ns1", "creds"),
		smbPV("pv2", "vol2", "ns2", "creds"),
		&v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "other"},
			Spec: v1.PersistentVolumeSpec{PersistentVolumeSource: v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{
				Driver: "other.csi.k8s.io", VolumeHandle: "vol2", NodeStageSecretRef: &v1.SecretReference{Namespace: "ns3", Name: "creds"},
			}}},
		},
		keytabSecret("ns1", "creds", testPrincipal, "bad old keytab"),
		keytabSecret("ns2", "creds", testPrincipal, "new keytab"),
		keytabSecret("ns3", "creds", testPrincipal, "keytab of another driver"),
	)
	// staged before the password change ("keytab" is revoked from now on)
	revokedKinit := kinitFunc
	kinitFunc = func(ctx context.Context, keytabPath, principal, ccachePath string) error {
		return os.WriteFile(ccachePath, []byte("staged"), 0600)
	}
	for _, volumeID := range []string{"vol1", "vol2"} {
		if _, _, err := d.ensureKerberosKeytab(volumeID, "/stage/"+volumeID, false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab")); err != nil {
			t.Fatal(err)
		}
	}
	kinitFunc = revokedKinit
	credUID, _ := d.keytabVolumeCredUID(d.keytabVolumeFile("vol1"))

	if renewed := d.renewKerberosTickets(); renewed != 1 {
		t.Fatalf("renewed = %d, want 1", renewed)
	}
	if got := readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)); got != "new keytab" {
		t.Errorf("stored keytab = %q, want the working keytab of ns2/creds", got)
	}
	if v, _ := readKeytabVolume(d.keytabVolumeFile("vol2")); v.SecretNamespace != "ns2" {
		t.Errorf("volume file of vol2 = %+v, want the secret of the smb PersistentVolume", v)
	}
}

func TestKeytabFromSecretDisabled(t *testing.T) {
	d := newKeytabTestDriver(t)
	revoked := map[string]bool{}
	fakeKinitRejecting(t, revoked)
	d.Name = DefaultDriverName
	client := fake.NewSimpleClientset(smbPV("pv1", "vol1", "app", "creds"), keytabSecret("app", "creds", testPrincipal, "keytab2"))
	d.kubeClient = client
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab1")); err != nil {
		t.Fatal(err)
	}
	revoked["keytab1"] = true

	// disabled: no API calls, the renewal fails
	d.krb5UpdateKeytabFromSecret = false
	if renewed := d.renewKerberosTickets(); renewed != 0 {
		t.Errorf("renewed = %d, want 0", renewed)
	}
	if n := len(client.Actions()); n != 0 {
		t.Errorf("API calls = %d, want 0", n)
	}
	// enabled without a kubernetes client: the same
	d.krb5UpdateKeytabFromSecret = true
	d.kubeClient = nil
	if renewed := d.renewKerberosTickets(); renewed != 0 {
		t.Errorf("renewed = %d, want 0", renewed)
	}
	credUID, _ := d.keytabVolumeCredUID(d.keytabVolumeFile("vol1"))
	if got := readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)); got != "keytab1" {
		t.Errorf("stored keytab = %q, want keytab1", got)
	}
}
