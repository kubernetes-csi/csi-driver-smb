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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	mount "k8s.io/mount-utils"
)

// userdbFiles lists the userdb directory: "<name>" for a file and
// "<name> -> <target>" for a symlink.
func userdbFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if target, err := os.Readlink(filepath.Join(dir, name)); err == nil {
			name += " -> " + target
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files
}

const testOwner = "smb.csi.k8s.io"

func writeUserdbFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureUserdbRecord(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("kerberos is only supported on Linux")
	}
	dir := t.TempDir()
	// twice: the second call must leave the record alone
	for i := 0; i < 2; i++ {
		if err := ensureUserdbRecord(dir, 2131564146, testOwner); err != nil {
			t.Fatalf("ensureUserdbRecord: %v", err)
		}
	}
	want := []string{"2131564146.user -> csi-smb-krb5-2131564146.user", "csi-smb-krb5-2131564146.user"}
	if files := userdbFiles(t, dir); !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
	path := filepath.Join(dir, "csi-smb-krb5-2131564146.user")
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0644 {
		t.Errorf("the record must have mode 0644: %v, %v", info, err)
	}
	var record map[string]interface{}
	if err := json.Unmarshal([]byte(readFile(t, path)), &record); err != nil {
		t.Fatalf("the record is not valid JSON: %v", err)
	}
	wantRecord := map[string]interface{}{
		"userName":      "csi-smb-krb5-2131564146",
		"uid":           float64(2131564146),
		"gid":           float64(2131564146),
		"realName":      "csi-driver-smb kerberos credential uid (smb.csi.k8s.io)",
		"homeDirectory": "/nonexistent",
		"shell":         "/usr/sbin/nologin",
	}
	if !reflect.DeepEqual(record, wantRecord) {
		t.Errorf("record = %v, want %v", record, wantRecord)
	}

	// a record of the driver without its owner (an older version), and a
	// missing symlink, are replaced
	writeUserdbFile(t, dir, "csi-smb-krb5-2131564146.user", `{"userName":"csi-smb-krb5-2131564146","realName":"csi-driver-smb kerberos credential uid"}`)
	if err := os.Remove(filepath.Join(dir, "2131564146.user")); err != nil {
		t.Fatal(err)
	}
	if err := ensureUserdbRecord(dir, 2131564146, testOwner); err != nil {
		t.Fatal(err)
	}
	if files := userdbFiles(t, dir); !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
	if !strings.Contains(readFile(t, path), "(smb.csi.k8s.io)") {
		t.Errorf("the record was not replaced: %q", readFile(t, path))
	}

	// records that are not of the driver are kept: another <uid>.user ...
	writeUserdbFile(t, dir, "alice.user", "{}")
	if err := os.Symlink("alice.user", filepath.Join(dir, "1500.user")); err != nil {
		t.Fatal(err)
	}
	writeUserdbFile(t, dir, "1600.user", "{}")
	// ... and the record of another instance of the driver
	if err := ensureUserdbRecord(dir, 1700, "smb2.csi.k8s.io"); err != nil {
		t.Fatal(err)
	}
	other := readFile(t, filepath.Join(dir, "csi-smb-krb5-1700.user"))
	for _, uid := range []int{1500, 1600, 1700} {
		if err := ensureUserdbRecord(dir, uid, testOwner); err != nil {
			t.Errorf("uid %d: %v", uid, err)
		}
	}
	want = []string{"1500.user -> alice.user", "1600.user", "1700.user -> csi-smb-krb5-1700.user", "2131564146.user -> csi-smb-krb5-2131564146.user", "alice.user", "csi-smb-krb5-1700.user", "csi-smb-krb5-2131564146.user"}
	if files := userdbFiles(t, dir); !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
	if got := readFile(t, filepath.Join(dir, "csi-smb-krb5-1700.user")); got != other {
		t.Errorf("the record of another driver was replaced: %q", got)
	}

	// system uids are refused
	for _, uid := range []int{-1, 0, 999} {
		if err := ensureUserdbRecord(dir, uid, testOwner); err == nil || (uid >= 0 && !strings.Contains(err.Error(), "system uid")) {
			t.Errorf("uid %d: err = %v, want a system uid error", uid, err)
		}
	}
	if err := ensureUserdbRecord(filepath.Join(dir, "missing"), 1000, testOwner); err == nil {
		t.Error("expected an error for a missing directory")
	}
}

func TestRemoveAndPruneUserdbRecords(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("kerberos is only supported on Linux")
	}
	dir := t.TempDir()
	for _, uid := range []int{1000, 2000, 3000} {
		if err := ensureUserdbRecord(dir, uid, testOwner); err != nil {
			t.Fatal(err)
		}
	}
	// records of another instance of the driver
	for _, uid := range []int{5000, 6000} {
		if err := ensureUserdbRecord(dir, uid, "smb2.csi.k8s.io"); err != nil {
			t.Fatal(err)
		}
	}
	// a record of an older version of the driver, without owner
	writeUserdbFile(t, dir, "csi-smb-krb5-7000.user", `{"realName":"csi-driver-smb kerberos credential uid"}`)
	// records that are not of the driver
	writeUserdbFile(t, dir, "alice.user", "{}")
	writeUserdbFile(t, dir, "csi-smb-krb5-x.user", "{}")
	if err := os.Symlink("alice.user", filepath.Join(dir, "4000.user")); err != nil {
		t.Fatal(err)
	}

	if err := removeUserdbRecord(dir, 1000, testOwner); err != nil {
		t.Fatal(err)
	}
	if err := removeUserdbRecord(dir, 1000, testOwner); err != nil {
		t.Errorf("removing a removed record: %v", err)
	}
	// the uid symlink of a record that is not of the driver is kept, and so
	// is the record of another instance
	for _, uid := range []int{4000, 5000} {
		if err := removeUserdbRecord(dir, uid, testOwner); err != nil {
			t.Fatal(err)
		}
	}
	pruneUserdbRecords(dir, map[int]bool{3000: true}, testOwner)

	want := []string{
		"3000.user -> csi-smb-krb5-3000.user", "4000.user -> alice.user",
		"5000.user -> csi-smb-krb5-5000.user", "6000.user -> csi-smb-krb5-6000.user",
		"alice.user", "csi-smb-krb5-3000.user", "csi-smb-krb5-5000.user", "csi-smb-krb5-6000.user", "csi-smb-krb5-x.user",
	}
	if files := userdbFiles(t, dir); !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

func TestRestoreKerberosUserEntriesUserdb(t *testing.T) {
	d := newKeytabTestDriver(t)
	d.krb5EnsurePasswdEntry = false
	d.krb5HostUserdbDirectory = t.TempDir()
	// a ticket cache, the record of a credential uid without a ticket cache,
	// and one of another instance of the driver
	if err := os.WriteFile(filepath.Join(d.krb5CacheDirectory, "krb5cc_1000"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureUserdbRecord(d.krb5HostUserdbDirectory, 2000, d.Name); err != nil {
		t.Fatal(err)
	}
	if err := ensureUserdbRecord(d.krb5HostUserdbDirectory, 3000, "smb2.csi.k8s.io"); err != nil {
		t.Fatal(err)
	}

	d.restoreKerberosUserEntries()
	want := []string{"1000.user -> csi-smb-krb5-1000.user", "3000.user -> csi-smb-krb5-3000.user", "csi-smb-krb5-1000.user", "csi-smb-krb5-3000.user"}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); !reflect.DeepEqual(files, want) {
		t.Errorf("files = %v, want %v", files, want)
	}
}

func TestKerberosTicketCacheUserdb(t *testing.T) {
	d := newKeytabTestDriver(t)
	credUID := os.Getuid()
	if credUID < krb5MinUserdbUID {
		t.Skipf("the tests run as uid %d, a system uid", credUID)
	}
	d.krb5EnsurePasswdEntry = false
	d.krb5HostUserdbDirectory = t.TempDir()
	mountFlags := []string{"sec=krb5", fmt.Sprintf("cruid=%d", credUID)}
	cacheKey := fmt.Sprintf("%s%d", d.krb5Prefix, credUID)

	// a secret without the ticket cache does not leave a record behind
	if _, err := d.ensureKerberosCache(d.krb5CacheDirectory, d.krb5Prefix, "vol1", mountFlags, map[string]string{}); err == nil {
		t.Fatal("expected an error for a secret without ticket cache")
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 0 {
		t.Errorf("userdb files = %v, want none", files)
	}

	// two volumes with the cruid: the record stays until the krb5cc_<cruid>
	// symlink is gone
	for _, volumeID := range []string{"vol1", "vol2"} {
		if _, err := d.ensureKerberosCache(d.krb5CacheDirectory, d.krb5Prefix, volumeID, mountFlags, map[string]string{cacheKey: "Y2FjaGU="}); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{fmt.Sprintf("%d.user -> csi-smb-krb5-%d.user", credUID, credUID), fmt.Sprintf("csi-smb-krb5-%d.user", credUID)}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); !reflect.DeepEqual(files, want) {
		t.Fatalf("userdb files = %v, want %v", files, want)
	}
	// krb5cc_<cruid> points to the cache of vol2
	if err := d.releaseKerberosCache("vol1"); err != nil {
		t.Fatal(err)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); !reflect.DeepEqual(files, want) {
		t.Errorf("userdb files after releasing vol1 = %v, want %v", files, want)
	}
	if err := d.releaseKerberosCache("vol2"); err != nil {
		t.Fatal(err)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 0 {
		t.Errorf("userdb files after releasing vol2 = %v, want none", files)
	}
}

func TestWithKerberosUpcallTarget(t *testing.T) {
	d := NewFakeDriver()
	flags := []string{"sec=krb5i", "vers=3.0"}
	if got := d.withKerberosUpcallTarget(flags); !reflect.DeepEqual(got, flags) {
		t.Errorf("without userdb directory: %v, want %v", got, flags)
	}

	d.krb5HostUserdbDirectory = "/run/userdb"
	got := d.withKerberosUpcallTarget(flags)
	if want := []string{"sec=krb5i", "vers=3.0", "upcall_target=mount"}; !reflect.DeepEqual(got, want) {
		t.Errorf("with userdb directory: %v, want %v", got, want)
	}
	if len(flags) != 2 {
		t.Errorf("the mount flags of the caller were changed: %v", flags)
	}
	for _, flags := range [][]string{
		{"sec=krb5", "upcall_target=app"},
		{"sec=krb5,upcall_target=app"},
		{"sec=krb5, upcall_target=mount"},
	} {
		if got := d.withKerberosUpcallTarget(flags); !reflect.DeepEqual(got, flags) {
			t.Errorf("an upcall_target of the mount flags must be kept: %v, want %v", got, flags)
		}
	}
}

// optionsMounter records the options of the mounts.
type optionsMounter struct {
	fakeMounter
	mu      sync.Mutex
	options map[string][]string
}

func (m *optionsMounter) MountSensitive(source, target, fstype string, options, sensitiveOptions []string) error {
	m.mu.Lock()
	m.options[target] = append([]string{}, options...)
	m.mu.Unlock()
	return m.fakeMounter.MountSensitive(source, target, fstype, options, sensitiveOptions)
}

func (m *optionsMounter) Mount(source, target, fstype string, options []string) error {
	return m.MountSensitive(source, target, fstype, options, nil)
}

func TestNodeStageVolumeWithHostUserdb(t *testing.T) {
	d := newKeytabTestDriver(t)
	fail := false
	fakeKinit(t, &fail)
	mounter := &optionsMounter{options: map[string][]string{}}
	d.mounter = &mount.SafeFormatAndMount{Interface: mounter}
	d.krb5HostUserdbDirectory = t.TempDir()

	stage := func(volumeID, targetPath string, mountFlags []string, secrets map[string]string) error {
		t.Helper()
		_, err := d.NodeStageVolume(context.Background(), &csi.NodeStageVolumeRequest{
			VolumeId:          volumeID,
			StagingTargetPath: targetPath,
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{MountFlags: mountFlags}},
				AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER},
			},
			VolumeContext: map[string]string{sourceField: "//server/share"},
			Secrets:       secrets,
		})
		return err
	}
	hasOption := func(target, option string) bool {
		for _, o := range mounter.options[target] {
			if o == option {
				return true
			}
		}
		return false
	}

	// a volume mounted with a keytab gets upcall_target=mount and a userdb record for its cruid
	target := filepath.Join(t.TempDir(), "stage")
	if err := stage("vol-keytab", target, []string{"sec=krb5i"}, keytabSecrets(testPrincipal, "keytab")); err != nil {
		t.Fatalf("NodeStageVolume: %v", err)
	}
	if !hasOption(target, "upcall_target=mount") {
		t.Errorf("mount options = %v, want upcall_target=mount", mounter.options[target])
	}
	credUID := credUIDOfFlags(t, mounter.options[target])
	want := []string{fmt.Sprintf("%d.user -> csi-smb-krb5-%d.user", credUID, credUID), fmt.Sprintf("csi-smb-krb5-%d.user", credUID)}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); !reflect.DeepEqual(files, want) {
		t.Errorf("userdb files = %v, want %v", files, want)
	}

	// a volume without kerberos does not
	plainTarget := filepath.Join(t.TempDir(), "stage")
	if err := stage("vol-plain", plainTarget, []string{"vers=3.0"}, map[string]string{"username": "user", "password": "pass"}); err != nil {
		t.Fatalf("NodeStageVolume: %v", err)
	}
	if hasOption(plainTarget, "upcall_target=mount") {
		t.Errorf("mount options = %v, want no upcall_target", mounter.options[plainTarget])
	}

	// unstaging the last volume of the principal removes the record
	if _, err := d.NodeUnstageVolume(context.Background(), &csi.NodeUnstageVolumeRequest{VolumeId: "vol-keytab", StagingTargetPath: target}); err != nil {
		t.Fatalf("NodeUnstageVolume: %v", err)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 0 {
		t.Errorf("userdb files after unstage = %v, want none", files)
	}

	// staging fails, and leaves nothing behind, when the record can not be written
	d.krb5HostUserdbDirectory = filepath.Join(t.TempDir(), "missing")
	err := stage("vol-keytab", filepath.Join(t.TempDir(), "stage"), []string{"sec=krb5i"}, keytabSecrets(testPrincipal, "keytab"))
	if err == nil || !strings.Contains(err.Error(), "userdb record") {
		t.Fatalf("expected staging to fail on the userdb record, got: %v", err)
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after a failed stage = %v, want none", files)
	}
}

func TestReleaseKerberosCacheKeepsOtherVolumes(t *testing.T) {
	d := newKeytabTestDriver(t)
	credUID := os.Getuid()
	if credUID < krb5MinUserdbUID {
		t.Skipf("the tests run as uid %d, a system uid", credUID)
	}
	d.krb5EnsurePasswdEntry = false
	d.krb5HostUserdbDirectory = t.TempDir()
	mountFlags := []string{"sec=krb5", fmt.Sprintf("cruid=%d", credUID)}
	secrets := map[string]string{fmt.Sprintf("%s%d", d.krb5Prefix, credUID): "Y2FjaGU="}
	for _, volumeID := range []string{"vol1", "vol2"} {
		if _, err := d.ensureKerberosCache(d.krb5CacheDirectory, d.krb5Prefix, volumeID, mountFlags, secrets); err != nil {
			t.Fatal(err)
		}
	}
	link := d.kerberosCacheFile(credUID)
	vol1Cache := getKerberosFilePath(d.krb5CacheDirectory, volumeKerberosCacheName("vol1"))

	// krb5cc_<cruid> points to the cache of vol2: releasing vol2 points it
	// to the cache of vol1, which keeps its record
	if err := d.releaseKerberosCache("vol2"); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(link); err != nil || target != vol1Cache {
		t.Errorf("%s -> %q (%v), want %q", link, target, err, vol1Cache)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 2 {
		t.Errorf("userdb files = %v, want the record of cruid %d", files, credUID)
	}
	if err := d.releaseKerberosCache("vol1"); err != nil {
		t.Fatal(err)
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files = %v, want none", files)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 0 {
		t.Errorf("userdb files = %v, want none", files)
	}
}

func TestNodeStageVolumeTicketCacheFailure(t *testing.T) {
	d := newKeytabTestDriver(t)
	credUID := os.Getuid()
	if credUID < krb5MinUserdbUID {
		t.Skipf("the tests run as uid %d, a system uid", credUID)
	}
	d.krb5EnsurePasswdEntry = false
	d.krb5HostUserdbDirectory = t.TempDir()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatal(err)
	}
	d.mounter = mounter
	secrets := map[string]string{fmt.Sprintf("%s%d", d.krb5Prefix, credUID): "Y2FjaGU="}
	stage := func(volumeID, source string) error {
		t.Helper()
		_, err := d.NodeStageVolume(context.Background(), &csi.NodeStageVolumeRequest{
			VolumeId:          volumeID,
			StagingTargetPath: filepath.Join(t.TempDir(), "stage"),
			VolumeCapability: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{MountFlags: []string{"sec=krb5", fmt.Sprintf("cruid=%d", credUID)}}},
				AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER},
			},
			VolumeContext: map[string]string{sourceField: source},
			Secrets:       secrets,
		})
		return err
	}
	failingSource := "//server/error_mount_sens_share"

	// a volume that can not be mounted leaves no ticket cache and no record
	if err := stage("vol-failed", failingSource); err == nil {
		t.Fatal("expected the mount to fail")
	}
	if files := kerberosFiles(t, d); len(files) != 0 {
		t.Errorf("files after a failed mount = %v, want none", files)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 0 {
		t.Errorf("userdb files after a failed mount = %v, want none", files)
	}

	// nor does it break a mounted volume with the same cruid
	if err := stage("vol-ok", "//server/share"); err != nil {
		t.Fatalf("NodeStageVolume: %v", err)
	}
	if err := stage("vol-failed", failingSource); err == nil {
		t.Fatal("expected the mount to fail")
	}
	okCache := getKerberosFilePath(d.krb5CacheDirectory, volumeKerberosCacheName("vol-ok"))
	if target, err := os.Readlink(d.kerberosCacheFile(credUID)); err != nil || target != okCache {
		t.Errorf("krb5cc -> %q (%v), want the cache of vol-ok %q", target, err, okCache)
	}
	if _, err := os.Stat(getKerberosFilePath(d.krb5CacheDirectory, volumeKerberosCacheName("vol-failed"))); !os.IsNotExist(err) {
		t.Errorf("the cache of the failed volume was kept: %v", err)
	}
	if files := userdbFiles(t, d.krb5HostUserdbDirectory); len(files) != 2 {
		t.Errorf("userdb files = %v, want the record of cruid %d", files, credUID)
	}
}
