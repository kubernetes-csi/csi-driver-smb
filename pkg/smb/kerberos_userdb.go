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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"k8s.io/klog/v2"
)

// The kernel gets the ticket for a session with an SMB server (when the
// volume is mounted, and again when the connection was lost) with
// request-key, which it starts in the namespaces of the host, and which runs
// cifs.upcall of the host. The key description holds the cruid, the pid of
// the process whose request needed the session, and the upcall_target mount
// option:
//
//   - upcall_target=app (the default): cifs.upcall switches to the
//     namespaces of that process before it reads the ticket cache. At mount
//     time that is mount.cifs in the smb container, which has the ticket
//     caches and the passwd entry of the cruid (ensurePasswdEntry). After a
//     reconnect it is a kernel thread (host namespaces, where the cruid is
//     unknown) or a process of a pod (no ticket caches): the session fails
//     and the volume stays unavailable.
//   - upcall_target=mount: cifs.upcall stays in the namespaces of the host,
//     those in which it was started; the kernel does not keep track of the
//     process that mounted the volume.
//
// With a directory of the host for systemd userdb drop-in records
// (--krb5-host-userdb-directory, /run/userdb of the host), the driver makes
// the cruid users known on the host with a user record in that directory,
// which nss-systemd reads, and mounts kerberos volumes with
// upcall_target=mount. The host must look for the ticket caches in the
// kerberos cache directory of the driver (krb5AutoConfig in the Helm chart).
// cifs-utils 7.7 and later look the cruid up on the host before switching
// namespaces, also with upcall_target=app: with these versions on the host
// the record is needed to mount the volume at all.
const (
	krb5UserNamePrefix   = "csi-smb-krb5-"
	userdbRecordSuffix   = ".user"
	upcallTargetOption   = "upcall_target="
	upcallTargetMountOpt = upcallTargetOption + "mount"

	// krb5UserdbRealName is the real name of the user records of the driver,
	// followed by the driver name (userdbRealName).
	krb5UserdbRealName = "csi-driver-smb kerberos credential uid"

	// krb5MinUserdbUID is the lowest credential uid that gets a user record:
	// the uids below it are system uids of the host.
	krb5MinUserdbUID = 1000
)

// userdbMutex serializes updates of the userdb directory.
var userdbMutex sync.Mutex

// kerberosUserName returns the name of the user of a credential uid in the
// passwd file of the smb container and in the userdb records of the host.
func kerberosUserName(uid int) string {
	return krb5UserNamePrefix + strconv.Itoa(uid)
}

// userdbRealName returns the real name of the user records of a driver: it
// tells the records of the driver apart from those of other instances of
// csi-driver-smb on the node (another driver name), which use the same user
// names.
func userdbRealName(owner string) string {
	return fmt.Sprintf("%s (%s)", krb5UserdbRealName, owner)
}

// userdbRecord returns the systemd user record (JSON User Records) of a
// credential uid of the driver owner.
func userdbRecord(uid int, owner string) ([]byte, error) {
	record, err := json.Marshal(map[string]interface{}{
		"userName":      kerberosUserName(uid),
		"uid":           uid,
		"gid":           uid,
		"realName":      userdbRealName(owner),
		"homeDirectory": "/nonexistent",
		"shell":         "/usr/sbin/nologin",
	})
	if err != nil {
		return nil, err
	}
	return append(record, '\n'), nil
}

// userdbRecordOwned tells whether the record file path is one of the driver
// owner, or of a version of the driver that did not record its owner.
func userdbRecordOwned(path, owner string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var record struct {
		RealName string `json:"realName"`
	}
	if err := json.Unmarshal(content, &record); err != nil {
		return false
	}
	return record.RealName == userdbRealName(owner) || record.RealName == krb5UserdbRealName
}

// ensureUserdbRecord makes sure that uid has a user record in the userdb
// directory dir: <name>.user, and the symlink <uid>.user to it for lookups
// by uid.
//
// A uid that already has a record of someone else (another <uid>.user, or a
// record of another instance of the driver) is left alone: it is known on
// the host already. System uids (below krb5MinUserdbUID) are refused: a
// record for them could shadow or confuse the users of the host.
func ensureUserdbRecord(dir string, uid int, owner string) error {
	if uid < krb5MinUserdbUID {
		return fmt.Errorf("cruid %d is a system uid, use a cruid of %d or more", uid, krb5MinUserdbUID)
	}
	record, err := userdbRecord(uid, owner)
	if err != nil {
		return err
	}
	name := kerberosUserName(uid) + userdbRecordSuffix
	recordPath := filepath.Join(dir, name)
	linkPath := filepath.Join(dir, strconv.Itoa(uid)+userdbRecordSuffix)

	userdbMutex.Lock()
	defer userdbMutex.Unlock()

	if _, err := os.Lstat(linkPath); err == nil {
		if target, err := os.Readlink(linkPath); err != nil || target != name {
			klog.V(2).Infof("uid %d has a userdb record in %s that is not of the driver, keeping it", uid, linkPath)
			return nil
		}
	}
	existing, err := os.ReadFile(recordPath)
	if err == nil && !userdbRecordOwned(recordPath, owner) {
		klog.V(2).Infof("uid %d has a userdb record in %s of another driver, keeping it", uid, recordPath)
		return nil
	}

	changed := false
	if err != nil || !bytes.Equal(existing, record) {
		// readable by everyone, like /etc/passwd
		temp := recordPath + tempFileSuffix()
		if err := os.WriteFile(temp, record, 0644); err != nil {
			os.Remove(temp)
			return err
		}
		if err := os.Chmod(temp, 0644); err != nil { // the umask may have removed bits
			os.Remove(temp)
			return err
		}
		if err := os.Rename(temp, recordPath); err != nil {
			os.Remove(temp)
			return err
		}
		changed = true
	}
	if target, err := os.Readlink(linkPath); err != nil || target != name {
		temp := linkPath + tempFileSuffix()
		if err := os.Symlink(name, temp); err != nil {
			return err
		}
		if err := os.Rename(temp, linkPath); err != nil {
			os.Remove(temp)
			return err
		}
		changed = true
	}
	if changed {
		klog.V(2).Infof("added the userdb record of kerberos credential uid %d to %s", uid, dir)
	}
	return nil
}

// removeUserdbRecord removes the user record of uid from the userdb
// directory dir, if it is one of the driver owner.
func removeUserdbRecord(dir string, uid int, owner string) error {
	userdbMutex.Lock()
	defer userdbMutex.Unlock()
	return removeUserdbRecordLocked(dir, uid, owner)
}

func removeUserdbRecordLocked(dir string, uid int, owner string) error {
	name := kerberosUserName(uid) + userdbRecordSuffix
	recordPath := filepath.Join(dir, name)
	if !userdbRecordOwned(recordPath, owner) {
		return nil
	}
	linkPath := filepath.Join(dir, strconv.Itoa(uid)+userdbRecordSuffix)
	if target, err := os.Readlink(linkPath); err == nil && target == name {
		if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// pruneUserdbRecords removes the user records of the driver owner from the
// userdb directory dir, except those of the uids in keep.
func pruneUserdbRecords(dir string, keep map[int]bool, owner string) {
	userdbMutex.Lock()
	defer userdbMutex.Unlock()
	records, err := filepath.Glob(filepath.Join(dir, krb5UserNamePrefix+"*"+userdbRecordSuffix))
	if err != nil {
		klog.Errorf("failed to list the userdb records in %s: %v", dir, err)
		return
	}
	for _, record := range records {
		uid, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(record), krb5UserNamePrefix), userdbRecordSuffix))
		if err != nil || uid < 0 || keep[uid] || !userdbRecordOwned(record, owner) {
			continue
		}
		if err := removeUserdbRecordLocked(dir, uid, owner); err != nil {
			klog.Errorf("failed to remove the userdb record of uid %d from %s: %v", uid, dir, err)
			continue
		}
		klog.V(2).Infof("removed the userdb record of kerberos credential uid %d from %s", uid, dir)
	}
}

// ensureKerberosUserEntries makes the user of a credential uid known to
// cifs.upcall: a passwd entry in the smb container (--krb5-ensure-passwd-entry)
// and a userdb record on the host (--krb5-host-userdb-directory).
//
// A failure to add the passwd entry is not fatal: the uid may be resolvable
// by other means. A failure to add the userdb record is, because the volume
// is mounted with upcall_target=mount and cifs.upcall then needs it.
func (d *Driver) ensureKerberosUserEntries(credUID int) error {
	if d.krb5EnsurePasswdEntry {
		if err := ensurePasswdEntry(d.passwdFile, credUID); err != nil {
			klog.Warningf("failed to ensure a passwd entry for kerberos credential uid %d in %s: %v", credUID, d.passwdFile, err)
		}
	}
	if d.krb5HostUserdbDirectory != "" {
		if err := ensureUserdbRecord(d.krb5HostUserdbDirectory, credUID, d.Name); err != nil {
			return fmt.Errorf("failed to add the userdb record of kerberos credential uid %d to %s: %v", credUID, d.krb5HostUserdbDirectory, err)
		}
	}
	return nil
}

// removeKerberosUserEntries removes the userdb record of a credential uid
// that is no longer used. The passwd entry in the smb container is kept: it
// is harmless and goes away with the container.
//
// The records of a uid used by volumes of two instances of the driver (the
// same principal) belong to the instance that added them first: when its
// last volume is unstaged the other instance loses the record until it
// stages a volume of the uid again or restarts.
func (d *Driver) removeKerberosUserEntries(credUID int) {
	if d.krb5HostUserdbDirectory == "" {
		return
	}
	if err := removeUserdbRecord(d.krb5HostUserdbDirectory, credUID, d.Name); err != nil {
		klog.Errorf("failed to remove the userdb record of kerberos credential uid %d from %s: %v", credUID, d.krb5HostUserdbDirectory, err)
	}
}

// releaseKerberosCache deletes the ticket cache of a volume mounted with a
// ticket cache in the secret (deleteKerberosCache). It is called when the
// volume is unstaged, and when staging a volume that was not staged yet
// fails.
//
// When the krb5cc_<cruid> symlink points to the cache of the volume, it is
// pointed to the cache of another volume with that cruid instead, when there
// is one, so that the other volumes keep a ticket cache. Otherwise it is
// removed with the cache, and so is the userdb record of the credential uid.
func (d *Driver) releaseKerberosCache(volumeID string) error {
	d.krb5Mutex.Lock()
	defer d.krb5Mutex.Unlock()

	volumeCache := getKerberosFilePath(d.krb5CacheDirectory, volumeKerberosCacheName(volumeID))
	var credUIDs []int
	links, _ := filepath.Glob(filepath.Join(d.krb5CacheDirectory, d.krb5Prefix+"*"))
	for _, link := range links {
		if target, err := os.Readlink(link); err != nil || target != volumeCache {
			continue
		}
		credUID, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(link), d.krb5Prefix))
		if err != nil {
			continue
		}
		if other := d.otherKerberosVolumeCache(credUID, volumeCache); other != "" {
			err := replaceSymlink(other, link)
			if err == nil {
				klog.V(2).Infof("pointed %s to the kerberos cache of another volume with cruid %d", link, credUID)
				continue
			}
			klog.Errorf("failed to point %s to %s: %v", link, other, err)
		}
		credUIDs = append(credUIDs, credUID)
	}
	if err := deleteKerberosCache(d.krb5CacheDirectory, volumeID); err != nil {
		return err
	}
	principals := d.keytabPrincipals()
	for _, credUID := range credUIDs {
		if _, usedByKeytab := principals[credUID]; !usedByKeytab && !d.kerberosCacheExists(credUID) {
			d.removeKerberosUserEntries(credUID)
		}
	}
	return nil
}

// otherKerberosVolumeCache returns the ticket cache file of another volume
// mounted with a ticket cache and credUID than the cache exclude: a regular
// file of the kerberos cache directory, owned by credUID, that is not a
// <krb5Prefix><uid> cache. It returns "" when there is none.
func (d *Driver) otherKerberosVolumeCache(credUID int, exclude string) string {
	entries, err := os.ReadDir(d.krb5CacheDirectory)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || strings.HasPrefix(entry.Name(), d.krb5Prefix) {
			continue
		}
		path := getKerberosFilePath(d.krb5CacheDirectory, entry.Name())
		if path == exclude {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if uid, ok := fileOwnerUID(info); ok && uid == credUID {
			return path
		}
	}
	return ""
}

// replaceSymlink makes link a symlink to target, replacing it atomically.
func replaceSymlink(target, link string) error {
	temp := link + tempFileSuffix()
	if err := os.Symlink(target, temp); err != nil {
		return err
	}
	if err := os.Rename(temp, link); err != nil {
		os.Remove(temp)
		return err
	}
	return nil
}

// restoreKerberosUserEntries adds the passwd entry and the userdb record of
// the credential uid of every ticket cache in the kerberos cache directory
// (<krb5Prefix><uid>), and removes the userdb records of credential uids that
// have no ticket cache any more.
//
// The passwd entries are added when a volume is staged, but they are part of
// the file system of the container: after a restart of the container they
// are gone while the volumes are still mounted. The userdb records are gone
// after a reboot of the node, but so are the mounts.
func (d *Driver) restoreKerberosUserEntries() {
	if !d.krb5EnsurePasswdEntry && d.krb5HostUserdbDirectory == "" {
		return
	}
	caches, err := filepath.Glob(filepath.Join(d.krb5CacheDirectory, d.krb5Prefix+"*"))
	if err != nil {
		klog.Errorf("failed to list kerberos ticket caches in %s: %v", d.krb5CacheDirectory, err)
		return
	}
	inUse := map[int]bool{}
	for _, cache := range caches {
		credUID, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(cache), d.krb5Prefix))
		if err != nil || credUID < 0 {
			continue
		}
		inUse[credUID] = true
		if err := d.ensureKerberosUserEntries(credUID); err != nil {
			klog.Error(err)
		}
	}
	if d.krb5HostUserdbDirectory != "" {
		pruneUserdbRecords(d.krb5HostUserdbDirectory, inUse, d.Name)
	}
}

// withKerberosUpcallTarget returns the mount flags of a kerberos mount with
// upcall_target=mount when the driver adds userdb records on the host, unless
// they already have an upcall_target.
func (d *Driver) withKerberosUpcallTarget(mountFlags []string) []string {
	if d.krb5HostUserdbDirectory == "" {
		return mountFlags
	}
	if hasMountOption(mountFlags, upcallTargetOption) {
		return mountFlags
	}
	return append(append([]string{}, mountFlags...), upcallTargetMountOpt)
}
