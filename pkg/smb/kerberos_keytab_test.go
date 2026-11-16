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
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	mount "k8s.io/mount-utils"
)

const (
	testPrincipal      = "user@EXAMPLE.COM"
	testOtherPrincipal = "other@EXAMPLE.COM"
)

// fakeKinit replaces kinit by a function that writes "<principal> <keytab>
// #<call>" into the cache, or fails when *fail is set, and makes chown a
// no-op so that the tests do not need root. It returns the call counter.
func fakeKinit(t *testing.T, fail *bool) *int {
	t.Helper()
	calls := 0
	origKinit, origChown := kinitFunc, chownFunc
	t.Cleanup(func() { kinitFunc, chownFunc = origKinit, origChown })
	chownFunc = func(string, int, int) error { return nil }
	kinitFunc = func(_ context.Context, keytabPath, principal, ccachePath string) error {
		calls++
		if *fail {
			return fmt.Errorf("kinit failed")
		}
		keytab, err := os.ReadFile(keytabPath)
		if err != nil {
			return err
		}
		return os.WriteFile(ccachePath, []byte(fmt.Sprintf("%s %s #%d", principal, keytab, calls)), 0600)
	}
	return &calls
}

func keytabSecrets(principal, keytab string) map[string]string {
	return map[string]string{
		krb5PrincipalSecretKey: principal,
		krb5KeytabSecretKey:    base64.StdEncoding.EncodeToString([]byte(keytab)),
	}
}

// newKeytabTestDriver returns a driver with an empty kerberos cache directory.
func newKeytabTestDriver(t *testing.T) *Driver {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("kerberos is only supported on Linux")
	}
	d := NewFakeDriver()
	d.krb5CacheDirectory = t.TempDir() + "/"
	return d
}

// kerberosFiles lists the files below the kerberos cache directory, with the
// name of the volume files replaced by the given names.
func kerberosFiles(t *testing.T, d *Driver, volumeIDs ...string) []string {
	t.Helper()
	names := map[string]string{}
	for _, volumeID := range volumeIDs {
		names[filepath.Base(d.keytabVolumeFile(volumeID))] = volumeID + krb5VolumeFileSuffix
	}
	files := []string{}
	err := filepath.Walk(d.krb5CacheDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(d.krb5CacheDirectory, path)
		if name, ok := names[filepath.Base(rel)]; ok {
			rel = filepath.Join(filepath.Dir(rel), name)
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	return files
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func credUIDOfFlags(t *testing.T, flags []string) int {
	t.Helper()
	credUID, err := getCredUID(flags)
	if err != nil {
		t.Fatalf("no cruid in the mount flags %v", flags)
	}
	return credUID
}

func TestGetKerberosKeytab(t *testing.T) {
	keytab := base64.StdEncoding.EncodeToString([]byte("keytab"))
	tests := []struct {
		desc      string
		secrets   map[string]string
		wantFound bool
		wantErr   string
	}{
		{desc: "no keytab", secrets: map[string]string{"username": "user", "krb5cc_1000": "Y2FjaGU="}},
		{desc: "keytab and principal", secrets: map[string]string{"krb5keytab": keytab, "krb5principal": " user@EXAMPLE.COM\n"}, wantFound: true},
		{desc: "keys are not case sensitive", secrets: map[string]string{"Krb5Keytab": keytab, "KRB5PRINCIPAL": testPrincipal}, wantFound: true},
		{desc: "keytab without principal", secrets: map[string]string{"krb5keytab": keytab}, wantErr: "both krb5keytab and krb5principal must be set"},
		{desc: "principal without keytab", secrets: map[string]string{"krb5principal": testPrincipal}, wantErr: "both krb5keytab and krb5principal must be set"},
		{desc: "keytab is not base64", secrets: map[string]string{"krb5keytab": "not base64!", "krb5principal": testPrincipal}, wantErr: "malformed kerberos keytab"},
		{desc: "principal looks like an option", secrets: map[string]string{"krb5keytab": keytab, "krb5principal": "-V"}, wantErr: "invalid kerberos principal"},
		{desc: "principal with whitespace", secrets: map[string]string{"krb5keytab": keytab, "krb5principal": "user @EXAMPLE.COM"}, wantErr: "invalid kerberos principal"},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			principal, content, found, err := getKerberosKeytab(test.secrets)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("err = %v, want an error containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil || found != test.wantFound {
				t.Fatalf("found = %t, err = %v, want found = %t", found, err, test.wantFound)
			}
			if found && (principal != testPrincipal || string(content) != "keytab") {
				t.Errorf("principal = %q, keytab = %q", principal, content)
			}
		})
	}
}

func TestEnsureKerberosKeytabSkipped(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	calls := fakeKinit(t, &fail)

	// not a kerberos mount
	flags, used, err := d.ensureKerberosKeytab("vol", "", false, []string{"vers=3.0"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil || used || !reflect.DeepEqual(flags, []string{"vers=3.0"}) {
		t.Errorf("mount without sec=krb5: flags = %v, used = %t, err = %v", flags, used, err)
	}
	// a kerberos mount with a ticket cache in the secret
	_, used, err = d.ensureKerberosKeytab("vol", "", false, []string{"sec=krb5", "cruid=1000"}, map[string]string{"krb5cc_1000": "Y2FjaGU="})
	if err != nil || used {
		t.Errorf("secret without keytab: used = %t, err = %v", used, err)
	}
	if *calls != 0 || len(kerberosFiles(t, d)) != 0 {
		t.Errorf("kinit calls = %d, files = %v, want none", *calls, kerberosFiles(t, d))
	}
}

func TestKerberosKeytabLifecycle(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	calls := fakeKinit(t, &fail)
	passwdFile := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(passwdFile, []byte("root:x:0:0:root:/root:/bin/bash\n"), 0644); err != nil {
		t.Fatal(err)
	}
	d.passwdFile = passwdFile
	d.krb5EnsurePasswdEntry = true

	// the first volume of a principal: the driver picks the cruid
	mountFlags := []string{"sec=krb5i", "vers=3.0"}
	flags, used, err := d.ensureKerberosKeytab("vol1", "", false, mountFlags, keytabSecrets(testPrincipal, "keytab1"))
	if err != nil || !used {
		t.Fatalf("ensureKerberosKeytab: used = %t, err = %v", used, err)
	}
	credUID := credUIDOfFlags(t, flags)
	if credUID < krb5AllocatedUIDBase || credUID >= krb5AllocatedUIDBase+krb5AllocatedUIDCount {
		t.Errorf("cruid %d is outside of the allocated range", credUID)
	}
	if !reflect.DeepEqual(flags[:2], mountFlags) || len(mountFlags) != 2 {
		t.Errorf("flags = %v, want %v with the cruid appended", flags, mountFlags)
	}
	uid := fmt.Sprint(credUID)
	wantFiles := []string{"keytabs/" + uid + ".keytab", "keytabs/" + uid + ".principal", "keytabs/vol1.volume", "krb5cc_" + uid}
	if files := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(files, wantFiles) {
		t.Errorf("files = %v, want %v", files, wantFiles)
	}
	if cache := readFile(t, d.kerberosCacheFile(credUID)); cache != testPrincipal+" keytab1 #1" {
		t.Errorf("ticket cache = %q", cache)
	}
	for _, file := range []string{d.keytabFile(credUID, krb5KeytabFileSuffix), d.kerberosCacheFile(credUID)} {
		if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0600 {
			t.Errorf("%s must have mode 0600: %v, %v", file, info, err)
		}
	}
	if passwd := readFile(t, passwdFile); !strings.Contains(passwd, ":x:"+uid+":") {
		t.Errorf("no passwd entry for cruid %s: %q", uid, passwd)
	}

	// a second volume of the principal shares the cruid; its (new) keytab replaces the stored one
	flags, _, err = d.ensureKerberosKeytab("vol2", "", false, mountFlags, keytabSecrets(testPrincipal, "keytab2"))
	if err != nil || credUIDOfFlags(t, flags) != credUID {
		t.Fatalf("second volume: flags = %v, err = %v, want cruid %d", flags, err, credUID)
	}
	if cache := readFile(t, d.kerberosCacheFile(credUID)); cache != testPrincipal+" keytab2 #2" {
		t.Errorf("ticket cache = %q", cache)
	}
	if keytab := readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)); keytab != "keytab2" {
		t.Errorf("stored keytab = %q", keytab)
	}

	// staging a volume again is fine
	if _, _, err = d.ensureKerberosKeytab("vol2", "", false, flags, keytabSecrets(testPrincipal, "keytab2")); err != nil {
		t.Errorf("staging again: %v", err)
	}
	if *calls != 3 {
		t.Errorf("kinit calls = %d, want 3", *calls)
	}

	// the keytab and the ticket cache stay until the last volume is released
	d.releaseKerberosKeytab("vol1")
	wantFiles[2] = "keytabs/vol2.volume"
	if files := kerberosFiles(t, d, "vol1", "vol2"); !reflect.DeepEqual(files, wantFiles) {
		t.Errorf("files after releasing vol1 = %v, want %v", files, wantFiles)
	}
	d.releaseKerberosKeytab("vol1")
	d.releaseKerberosKeytab("not-staged")
	d.releaseKerberosKeytab("vol2")
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after releasing all volumes = %v, want none", files)
	}
}

func TestKerberosKeytabCredUIDPerPrincipal(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)
	mountFlags := []string{"sec=krb5"}

	stage := func(volumeID, principal string, flags ...string) (int, error) {
		t.Helper()
		flags, _, err := d.ensureKerberosKeytab(volumeID, "", false, append(mountFlags, flags...), keytabSecrets(principal, "keytab of "+principal))
		if err != nil {
			d.releaseKerberosKeytab(volumeID)
			return 0, err
		}
		return credUIDOfFlags(t, flags), nil
	}

	credUID, err := stage("vol1", testPrincipal)
	if err != nil {
		t.Fatal(err)
	}
	otherCredUID, err := stage("vol2", testOtherPrincipal)
	if err != nil {
		t.Fatal(err)
	}
	if credUID == otherCredUID {
		t.Fatalf("two principals share cruid %d", credUID)
	}
	for uid, principal := range map[int]string{credUID: testPrincipal, otherCredUID: testOtherPrincipal} {
		if cache := readFile(t, d.kerberosCacheFile(uid)); !strings.HasPrefix(cache, principal+" ") {
			t.Errorf("ticket cache of cruid %d = %q, want a ticket of %s", uid, cache, principal)
		}
	}

	// a cruid mount flag is used as it is
	if uid, err := stage("vol3", "third@EXAMPLE.COM", "cruid=1234"); err != nil || uid != 1234 {
		t.Errorf("explicit cruid: uid = %d, err = %v", uid, err)
	}
	if uid, err := stage("vol4", "third@EXAMPLE.COM"); err != nil || uid != 1234 {
		t.Errorf("volume of a principal with an explicit cruid: uid = %d, err = %v", uid, err)
	}
	// but not when another principal uses it
	before := kerberosFiles(t, d)
	if _, err := stage("vol5", testPrincipal, fmt.Sprintf("cruid=%d", otherCredUID)); err == nil || !strings.Contains(err.Error(), "another kerberos principal") {
		t.Errorf("cruid of another principal: err = %v", err)
	}
	// or volumes mounted with a ticket cache from their secret
	if err := os.Symlink("volume-cache", d.kerberosCacheFile(1000)); err != nil {
		t.Fatal(err)
	}
	if _, err := stage("vol5", testPrincipal, "cruid=1000"); err == nil || !strings.Contains(err.Error(), "kerberos ticket cache") {
		t.Errorf("cruid of a ticket cache: err = %v", err)
	}
	os.Remove(d.kerberosCacheFile(1000))
	if _, err := stage("vol5", testPrincipal, "cruid=abc"); err == nil {
		t.Error("expected an error for an invalid cruid")
	}
	if after := kerberosFiles(t, d); !reflect.DeepEqual(after, before) {
		t.Errorf("files after the refused mounts = %v, want %v", after, before)
	}
	if cache := readFile(t, d.kerberosCacheFile(otherCredUID)); !strings.HasPrefix(cache, testOtherPrincipal+" ") {
		t.Errorf("ticket cache of cruid %d was replaced: %q", otherCredUID, cache)
	}

	// a volume staged as another principal moves to the cruid of that principal
	if uid, err := stage("vol1", testOtherPrincipal); err != nil || uid != otherCredUID {
		t.Errorf("volume staged as another principal: uid = %d, err = %v", uid, err)
	}
	if d.kerberosCacheExists(credUID) {
		t.Errorf("the ticket cache of cruid %d was not removed", credUID)
	}
}

func TestKerberosKeytabCredUIDCollision(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)

	flags, _, err := d.ensureKerberosKeytab("vol1", "", false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil {
		t.Fatal(err)
	}
	credUID := credUIDOfFlags(t, flags)
	d.releaseKerberosKeytab("vol1")

	// the cruid the principal would get is taken by a ticket cache, the next one by another principal
	if err := os.WriteFile(d.kerberosCacheFile(credUID), []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.ensureKerberosKeytab("vol2", "", false, []string{"sec=krb5", fmt.Sprintf("cruid=%d", credUID+1)}, keytabSecrets(testOtherPrincipal, "keytab")); err != nil {
		t.Fatal(err)
	}
	flags, _, err = d.ensureKerberosKeytab("vol1", "", false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil || credUIDOfFlags(t, flags) != credUID+2 {
		t.Errorf("flags = %v, err = %v, want cruid %d", flags, err, credUID+2)
	}
}

func TestKerberosKeytabKinitFailure(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)
	mountFlags := []string{"sec=krb5"}

	flags, _, err := d.ensureKerberosKeytab("vol1", "", false, mountFlags, keytabSecrets(testPrincipal, "good keytab"))
	if err != nil {
		t.Fatal(err)
	}
	credUID := credUIDOfFlags(t, flags)
	before := kerberosFiles(t, d, "vol1")
	cache := readFile(t, d.kerberosCacheFile(credUID))

	// a keytab that does not work replaces neither the stored keytab nor the ticket cache
	fail = true
	for _, volumeID := range []string{"vol1", "vol2"} {
		if _, _, err := d.ensureKerberosKeytab(volumeID, "", false, mountFlags, keytabSecrets(testPrincipal, "bad keytab")); err == nil || !strings.Contains(err.Error(), "failed to get a kerberos ticket") {
			t.Fatalf("%s: err = %v, want a kinit failure", volumeID, err)
		}
	}
	// vol2 is not staged: NodeStageVolume releases it
	d.releaseKerberosKeytab("vol2")
	if after := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(after, before) {
		t.Errorf("files = %v, want %v", after, before)
	}
	if keytab := readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)); keytab != "good keytab" {
		t.Errorf("stored keytab = %q", keytab)
	}
	if got := readFile(t, d.kerberosCacheFile(credUID)); got != cache {
		t.Errorf("ticket cache = %q, want %q", got, cache)
	}

	// nothing is left of a principal whose only volume could not be staged
	if _, _, err := d.ensureKerberosKeytab("vol3", "", false, mountFlags, keytabSecrets(testOtherPrincipal, "bad keytab")); err == nil {
		t.Fatal("expected a kinit failure")
	}
	d.releaseKerberosKeytab("vol3")
	if after := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(after, before) {
		t.Errorf("files = %v, want %v", after, before)
	}
}

func TestNodeStageVolumeWithKerberosKeytab(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatal(err)
	}
	d.mounter = mounter

	stage := func(volumeID, source, targetPath string, secrets map[string]string) error {
		t.Helper()
		_, err := d.NodeStageVolume(context.Background(), &csi.NodeStageVolumeRequest{
			VolumeId:          volumeID,
			StagingTargetPath: targetPath,
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{MountFlags: []string{"sec=krb5i"}}},
				AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER},
			},
			VolumeContext: map[string]string{sourceField: source},
			Secrets:       secrets,
		})
		return err
	}
	unstage := func(volumeID, targetPath string) {
		t.Helper()
		if _, err := d.NodeUnstageVolume(context.Background(), &csi.NodeUnstageVolumeRequest{VolumeId: volumeID, StagingTargetPath: targetPath}); err != nil {
			t.Fatalf("NodeUnstageVolume: %v", err)
		}
	}
	secrets := keytabSecrets(testPrincipal, "keytab")
	failingSource := "//server/error_mount_sens_share"

	// the mount fails: nothing of the volume may stay behind
	if err := stage("vol-failed", failingSource, filepath.Join(t.TempDir(), "stage"), secrets); err == nil || !strings.Contains(err.Error(), "fake MountSensitive") {
		t.Fatalf("expected the mount to fail, got: %v", err)
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after a failed mount = %v, want none", files)
	}
	// also when staging fails before the mount
	if err := stage("vol-failed", "//server/share/../..", filepath.Join(t.TempDir(), "stage"), secrets); err == nil {
		t.Fatal("expected an invalid source path to be refused")
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after a failed stage = %v, want none", files)
	}

	// a staged volume of the principal keeps the keytab and the ticket cache
	// when another volume of the principal can not be mounted
	target := filepath.Join(t.TempDir(), "stage")
	if err := stage("vol-ok", "//server/share", target, secrets); err != nil {
		t.Fatalf("NodeStageVolume: %v", err)
	}
	staged := kerberosFiles(t, d, "vol-ok")
	if len(staged) != 4 {
		t.Fatalf("files of the staged volume = %v", staged)
	}
	if err := stage("vol-failed", failingSource, filepath.Join(t.TempDir(), "stage"), secrets); err == nil {
		t.Fatal("expected the mount to fail")
	}
	if files := kerberosFiles(t, d, "vol-ok"); !reflect.DeepEqual(files, staged) {
		t.Errorf("files = %v, want %v", files, staged)
	}

	// unstage removes them
	unstage("vol-ok", target)
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after unstage = %v, want none", files)
	}
}

func TestKerberosTicketCacheWithCredUIDOfKeytab(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)

	flags, _, err := d.ensureKerberosKeytab("vol1", "", false, []string{"sec=krb5", "cruid=1000"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil {
		t.Fatal(err)
	}
	cache := readFile(t, d.kerberosCacheFile(1000))
	_, err = d.ensureKerberosCache(d.krb5CacheDirectory, d.krb5Prefix, "vol2", flags, map[string]string{"krb5cc_1000": "Y2FjaGU="})
	if err == nil || !strings.Contains(err.Error(), "kerberos keytab") {
		t.Errorf("err = %v, want the cruid to be refused", err)
	}
	if got := readFile(t, d.kerberosCacheFile(1000)); got != cache {
		t.Errorf("ticket cache = %q, want %q", got, cache)
	}
}

func TestKerberosKeytabMountedVolume(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	calls := fakeKinit(t, &fail)
	mountFlags := []string{"sec=krb5"}

	flags, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", false, mountFlags, keytabSecrets(testPrincipal, "keytab"))
	if err != nil {
		t.Fatal(err)
	}
	credUID := credUIDOfFlags(t, flags)
	files := kerberosFiles(t, d, "vol1")
	cache := readFile(t, d.kerberosCacheFile(credUID))
	*calls = 0

	// a mounted volume keeps its principal and cruid when they change in the
	// secret or the mount flags: nothing changes on the node
	for _, test := range []struct {
		name       string
		mountFlags []string
		principal  string
	}{
		{"other principal", mountFlags, testOtherPrincipal},
		{"other principal and cruid", []string{"sec=krb5", "cruid=1000"}, testOtherPrincipal},
	} {
		flags, used, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, test.mountFlags, keytabSecrets(test.principal, "other keytab"))
		if err != nil || !used {
			t.Fatalf("%s: used = %t, err = %v", test.name, used, err)
		}
		if got := credUIDOfFlags(t, flags); got != credUID || len(flags) != 2 {
			t.Errorf("%s: flags = %v, want cruid %d", test.name, flags, credUID)
		}
		if got := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(got, files) {
			t.Errorf("%s: files = %v, want %v", test.name, got, files)
		}
		if got := readFile(t, d.keytabFile(credUID, krb5KeytabFileSuffix)); got != "keytab" {
			t.Errorf("%s: stored keytab = %q", test.name, got)
		}
		if got := readFile(t, d.kerberosCacheFile(credUID)); got != cache {
			t.Errorf("%s: ticket cache = %q, want %q", test.name, got, cache)
		}
	}
	if *calls != 0 {
		t.Errorf("kinit calls = %d, want 0 for another principal", *calls)
	}

	// another cruid: the volume keeps its cruid, and gets a new ticket
	flags, _, err = d.ensureKerberosKeytab("vol1", "/stage/vol1", true, []string{"sec=krb5", "cruid=1000"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil {
		t.Fatal(err)
	}
	if got := credUIDOfFlags(t, flags); got != credUID || len(flags) != 2 {
		t.Errorf("other cruid: flags = %v, want cruid %d", flags, credUID)
	}
	if got := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(got, files) {
		t.Errorf("other cruid: files = %v, want %v", got, files)
	}
	if *calls != 1 {
		t.Errorf("other cruid: kinit calls = %d, want 1", *calls)
	}
	cache = readFile(t, d.kerberosCacheFile(credUID))

	// a mounted volume keeps its ticket cache when no new ticket can be obtained
	fail = true
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, mountFlags, keytabSecrets(testPrincipal, "keytab")); err != nil {
		t.Errorf("mounted volume, kinit fails: %v", err)
	}
	if got := readFile(t, d.kerberosCacheFile(credUID)); got != cache {
		t.Errorf("ticket cache = %q, want %q", got, cache)
	}
	// but not without a ticket cache
	if err := os.Remove(d.kerberosCacheFile(credUID)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, mountFlags, keytabSecrets(testPrincipal, "keytab")); err == nil {
		t.Error("mounted volume without a ticket cache, kinit fails: expected an error")
	}
	// and not for a volume that is not mounted
	fail = false
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, mountFlags, keytabSecrets(testPrincipal, "keytab")); err != nil {
		t.Fatal(err)
	}
	fail = true
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", false, mountFlags, keytabSecrets(testPrincipal, "keytab")); err == nil {
		t.Error("volume not mounted, kinit fails: expected an error")
	}

	// a volume that is not mounted switches to the new principal
	fail = false
	flags, _, err = d.ensureKerberosKeytab("vol1", "/stage/vol1", false, mountFlags, keytabSecrets(testOtherPrincipal, "other keytab"))
	if err != nil {
		t.Fatal(err)
	}
	otherUID := credUIDOfFlags(t, flags)
	if otherUID == credUID {
		t.Fatalf("cruid = %d, want another one than %d", otherUID, credUID)
	}
	other := fmt.Sprint(otherUID)
	want := []string{"keytabs/" + other + ".keytab", "keytabs/" + other + ".principal", "keytabs/vol1.volume", "krb5cc_" + other}
	if got := kerberosFiles(t, d, "vol1"); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
}

func TestKeytabVolumeFile(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)

	flags, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil {
		t.Fatal(err)
	}
	credUID := credUIDOfFlags(t, flags)
	volumeFile := d.keytabVolumeFile("vol1")
	want := keytabVolume{CredUID: credUID, VolumeID: "vol1", StagingPath: "/stage/vol1"}
	if got, ok := readKeytabVolume(volumeFile); !ok || got != want {
		t.Errorf("volume = %+v, %t, want %+v", got, ok, want)
	}

	// the file of an older version of the driver only has the cruid; it is
	// rewritten when the mounted volume is staged again
	if err := os.WriteFile(volumeFile, []byte(fmt.Sprintf("%d\n", credUID)), 0600); err != nil {
		t.Fatal(err)
	}
	if got, ok := readKeytabVolume(volumeFile); !ok || got != (keytabVolume{CredUID: credUID}) {
		t.Errorf("old volume file = %+v, %t", got, ok)
	}
	if _, _, err := d.ensureKerberosKeytab("vol1", "/stage/vol1", true, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab")); err != nil {
		t.Fatal(err)
	}
	if got, ok := readKeytabVolume(volumeFile); !ok || got != want {
		t.Errorf("rewritten volume = %+v, %t, want %+v", got, ok, want)
	}
}

func TestWithCredUID(t *testing.T) {
	for _, test := range []struct {
		flags, want []string
	}{
		{[]string{"sec=krb5"}, []string{"sec=krb5", "cruid=7"}},
		{[]string{"sec=krb5", "cruid=7"}, []string{"sec=krb5", "cruid=7"}},
		{[]string{"cruid=1000", "sec=krb5"}, []string{"cruid=7", "sec=krb5"}},
		{[]string{"vers=3.0,cruid=1000", "sec=krb5"}, []string{"vers=3.0,cruid=7", "sec=krb5"}},
		{[]string{"sec=krb5,vers=3.0"}, []string{"sec=krb5,vers=3.0", "cruid=7"}},
	} {
		flags := append([]string{}, test.flags...)
		if got := withCredUID(flags, 7); !reflect.DeepEqual(got, test.want) {
			t.Errorf("withCredUID(%v) = %v, want %v", test.flags, got, test.want)
		}
		if !reflect.DeepEqual(flags, test.flags) {
			t.Errorf("withCredUID changed its argument: %v", flags)
		}
	}
}

func TestPruneKeytabVolumes(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)
	fakeMounter := mount.NewFakeMounter(nil)
	d.mounter = &mount.SafeFormatAndMount{Interface: fakeMounter}

	dir := t.TempDir()
	stagingPath := func(volumeID string) string { return filepath.Join(dir, volumeID) }
	for _, volumeID := range []string{"mounted", "unmounted", "removed", "busy", "old"} {
		if volumeID != "removed" {
			if err := os.MkdirAll(stagingPath(volumeID), 0750); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := d.ensureKerberosKeytab(volumeID, stagingPath(volumeID), false, []string{"sec=krb5"}, keytabSecrets(testPrincipal, "keytab")); err != nil {
			t.Fatal(err)
		}
	}
	fakeMounter.MountPoints = []mount.MountPoint{{Device: "//server/share", Path: stagingPath("mounted"), Type: "cifs"}}
	// a volume that is being staged or unstaged
	lockKey := fmt.Sprintf("%s-%s", "busy", stagingPath("busy"))
	if !d.volumeLocks.TryAcquire(lockKey) {
		t.Fatal("failed to acquire the volume lock")
	}
	// a volume staged by an older version of the driver
	credUID, _ := d.keytabVolumeCredUID(d.keytabVolumeFile("old"))
	if err := os.WriteFile(d.keytabVolumeFile("old"), []byte(fmt.Sprintf("%d\n", credUID)), 0600); err != nil {
		t.Fatal(err)
	}

	d.pruneKeytabVolumes()
	for volumeID, want := range map[string]bool{"mounted": true, "unmounted": false, "removed": false, "busy": true, "old": true} {
		if _, err := os.Stat(d.keytabVolumeFile(volumeID)); (err == nil) != want {
			t.Errorf("%s: volume file kept = %t, want %t", volumeID, err == nil, want)
		}
	}
	if d.volumeLocks.TryAcquire(lockKey) {
		t.Error("the lock of the busy volume was released")
	}
	d.volumeLocks.Release(lockKey)
	if !d.volumeLocks.TryAcquire(fmt.Sprintf("%s-%s", "unmounted", stagingPath("unmounted"))) {
		t.Error("the lock of a pruned volume was not released")
	}

	// the keytab goes with the last volume
	for _, volumeID := range []string{"mounted", "busy", "old"} {
		d.releaseKerberosKeytab(volumeID)
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files = %v, want none", files)
	}
}

func TestKerberosKeytabCredUIDInCombinedMountFlag(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)

	// a kerberos mount with the cruid in a mount flag with several options
	flags, used, err := d.ensureKerberosKeytab("vol1", "", false, []string{"vers=3.0,sec=krb5i,cruid=1000"}, keytabSecrets(testPrincipal, "keytab"))
	if err != nil || !used {
		t.Fatalf("used = %t, err = %v", used, err)
	}
	if want := []string{"vers=3.0,sec=krb5i,cruid=1000"}; !reflect.DeepEqual(flags, want) {
		t.Errorf("flags = %v, want %v", flags, want)
	}
	if !d.kerberosCacheExists(1000) {
		t.Error("no ticket cache for cruid 1000")
	}
	if _, _, err := d.ensureKerberosKeytab("vol2", "", false, []string{"sec=krb5,cruid=x"}, keytabSecrets(testPrincipal, "keytab")); err == nil {
		t.Error("expected an error for an invalid cruid")
	}
}
