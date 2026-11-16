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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"
)

// Volumes can be mounted with a kerberos keytab instead of a ticket cache:
// the secret then holds the keytab (krb5keytab, base64 encoded) and the
// principal to authenticate as (krb5principal), and the driver gets the
// ticket itself.
//
// cifs.upcall reads the ticket of a mount from the cache of its credential
// uid (cruid): <krb5CacheDirectory>/<krb5Prefix><cruid>. There is one cache
// per credential uid, so every principal gets a credential uid of its own,
// shared by all volumes mounted as that principal on the node. What the
// driver needs to get tickets for a principal is kept in the "keytabs"
// directory of the kerberos cache directory:
//
//	<cruid>.keytab     the keytab
//	<cruid>.principal  the principal
//	<hash>.volume      one file per volume staged with the keytab, holding
//	                   its cruid, volume id and staging path (JSON);
//	                   <hash> is the sha256 of the volume id
//
// The keytab, the principal and the ticket cache of a credential uid are
// removed when the last volume using it is unstaged.
const (
	krb5KeytabSecretKey    = "krb5keytab"
	krb5PrincipalSecretKey = "krb5principal"

	krb5KeytabDirectory     = "keytabs"
	krb5KeytabFileSuffix    = ".keytab"
	krb5PrincipalFileSuffix = ".principal"
	krb5VolumeFileSuffix    = ".volume"

	// Credential uids picked by the driver for mounts without a cruid mount
	// option are taken from [krb5AllocatedUIDBase,
	// krb5AllocatedUIDBase+krb5AllocatedUIDCount): a range that fits in a
	// signed 32 bit integer and is far above the uids that are normally in
	// use on a node.
	krb5AllocatedUIDBase  = 2000000000
	krb5AllocatedUIDCount = 147000000

	kinitTimeout = 30 * time.Second
)

// errKeytabReleased is returned when the last volume of a principal was
// unstaged while a ticket was being obtained for it.
var errKeytabReleased = errors.New("the kerberos keytab is no longer in use")

// kinitFunc gets a ticket for principal with the keytab keytabPath and
// writes it into the credential cache file ccachePath. It is a variable for
// testing.
var kinitFunc = func(ctx context.Context, keytabPath, principal, ccachePath string) error {
	cmd := exec.CommandContext(ctx, "kinit", "-k", "-t", keytabPath, "-c", "FILE:"+ccachePath, "--", principal)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kinit failed: %v, output: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// chownFunc is a variable for testing.
var chownFunc = os.Chown

// getKerberosKeytab returns the principal and the keytab of the secret.
// found is false when the secret has neither.
func getKerberosKeytab(secrets map[string]string) (principal string, keytab []byte, found bool, err error) {
	var encodedKeytab string
	for k, v := range secrets {
		switch strings.ToLower(k) {
		case krb5KeytabSecretKey:
			encodedKeytab = strings.TrimSpace(v)
		case krb5PrincipalSecretKey:
			principal = strings.TrimSpace(v)
		}
	}
	if encodedKeytab == "" && principal == "" {
		return "", nil, false, nil
	}
	if encodedKeytab == "" || principal == "" {
		return "", nil, false, fmt.Errorf("both %s and %s must be set to mount with a kerberos keytab", krb5KeytabSecretKey, krb5PrincipalSecretKey)
	}
	// the principal is passed to kinit and written to a file
	if strings.HasPrefix(principal, "-") || strings.ContainsAny(principal, " \t\r\n") {
		return "", nil, false, fmt.Errorf("invalid kerberos principal in key %s", krb5PrincipalSecretKey)
	}
	keytab, err = base64.StdEncoding.DecodeString(encodedKeytab)
	if err != nil {
		return "", nil, false, fmt.Errorf("malformed kerberos keytab in key %s, expected to be in base64 form: %v", krb5KeytabSecretKey, err)
	}
	if len(keytab) == 0 {
		return "", nil, false, fmt.Errorf("empty kerberos keytab in key %s", krb5KeytabSecretKey)
	}
	return principal, keytab, true, nil
}

func (d *Driver) keytabDirectory() string {
	return filepath.Join(d.krb5CacheDirectory, krb5KeytabDirectory)
}

// keytabFile returns the path of the keytab (krb5KeytabFileSuffix) or the
// principal (krb5PrincipalFileSuffix) of a credential uid.
func (d *Driver) keytabFile(credUID int, suffix string) string {
	return filepath.Join(d.keytabDirectory(), strconv.Itoa(credUID)+suffix)
}

// keytabVolumeFile returns the path of the file recording that a volume is
// staged with a keytab. The name is a hash of the volume id: a volume id can
// be too long for a file name.
func (d *Driver) keytabVolumeFile(volumeID string) string {
	sum := sha256.Sum256([]byte(volumeID))
	return filepath.Join(d.keytabDirectory(), hex.EncodeToString(sum[:])+krb5VolumeFileSuffix)
}

// kerberosCacheFile returns the path of the ticket cache of a credential uid.
func (d *Driver) kerberosCacheFile(credUID int) string {
	return filepath.Join(d.krb5CacheDirectory, getKrb5CcacheName(d.krb5Prefix, credUID))
}

// keytabPrincipals returns the principal of every credential uid that has
// volumes staged with a keytab.
func (d *Driver) keytabPrincipals() map[int]string {
	principals := map[int]string{}
	files, _ := filepath.Glob(filepath.Join(d.keytabDirectory(), "*"+krb5PrincipalFileSuffix))
	for _, file := range files {
		credUID, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(file), krb5PrincipalFileSuffix))
		if err != nil {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			klog.Errorf("failed to read kerberos principal %s: %v", file, err)
			continue
		}
		principals[credUID] = strings.TrimSpace(string(content))
	}
	return principals
}

// keytabVolume is the content of the file of a volume staged with a keytab.
type keytabVolume struct {
	CredUID     int    `json:"cruid"`
	VolumeID    string `json:"volumeID"`
	StagingPath string `json:"stagingPath"`
}

// readKeytabVolume reads the file of a volume staged with a keytab. The file
// of a volume staged by an older version of the driver only holds the
// credential uid.
func readKeytabVolume(volumeFile string) (keytabVolume, bool) {
	content, err := os.ReadFile(volumeFile)
	if err != nil {
		return keytabVolume{}, false
	}
	var volume keytabVolume
	if err := json.Unmarshal(content, &volume); err == nil {
		return volume, true
	}
	credUID, err := strconv.Atoi(strings.TrimSpace(string(content)))
	return keytabVolume{CredUID: credUID}, err == nil
}

// keytabVolumeCredUID returns the credential uid of a volume staged with a
// keytab.
func (d *Driver) keytabVolumeCredUID(volumeFile string) (int, bool) {
	volume, ok := readKeytabVolume(volumeFile)
	return volume.CredUID, ok
}

// writeKeytabVolume records that a volume is staged at stagingPath with the
// keytab of credUID.
func (d *Driver) writeKeytabVolume(volumeID, stagingPath string, credUID int) error {
	content, err := json.Marshal(keytabVolume{CredUID: credUID, VolumeID: volumeID, StagingPath: stagingPath})
	if err != nil {
		return err
	}
	return writeFileAtomic(d.keytabVolumeFile(volumeID), append(content, '\n'))
}

// keytabVolumeCount returns the number of volumes staged with the keytab of
// a credential uid.
func (d *Driver) keytabVolumeCount(credUID int) int {
	count := 0
	files, _ := filepath.Glob(filepath.Join(d.keytabDirectory(), "*"+krb5VolumeFileSuffix))
	for _, file := range files {
		if uid, ok := d.keytabVolumeCredUID(file); ok && uid == credUID {
			count++
		}
	}
	return count
}

// kerberosCacheExists tells whether there is a ticket cache for credUID.
func (d *Driver) kerberosCacheExists(credUID int) bool {
	_, err := os.Lstat(d.kerberosCacheFile(credUID))
	return err == nil
}

// reserveKeytabCredential picks the credential uid for a volume staged at
// stagingPath with the keytab of principal and records that the volume uses
// it: the cruid of the mount flags when there is one, otherwise the
// credential uid of the principal. It returns the credential uid and its
// principal.
//
// A volume that is already mounted (mounted) keeps the credential uid it was
// staged with, also when the principal or the cruid mount flag changed since:
// the kernel uses the cruid of the mount until the volume is mounted again.
// The returned principal is then the one of the mount.
func (d *Driver) reserveKeytabCredential(volumeID, stagingPath string, mounted bool, mountFlags []string, principal string) (int, string, error) {
	d.krb5Mutex.Lock()
	defer d.krb5Mutex.Unlock()

	principals := d.keytabPrincipals()
	credUID := -1
	if hasMountOption(mountFlags, "cruid=") {
		var err error
		if credUID, err = getCredUID(mountFlags); err != nil || credUID < 0 {
			return 0, "", fmt.Errorf("invalid cruid mount option: %v", err)
		}
	}

	volumeFile := d.keytabVolumeFile(volumeID)
	if mounted {
		if previous, ok := d.keytabVolumeCredUID(volumeFile); ok {
			if inUseBy, ok := principals[previous]; ok {
				if inUseBy != principal || (credUID >= 0 && credUID != previous) {
					klog.Warningf("volume(%s) is mounted as kerberos principal %s with cruid %d, it keeps using them until it is mounted again", volumeID, inUseBy, previous)
				}
				// rewrite the file, it may be one of an older version of the driver
				if err := d.writeKeytabVolume(volumeID, stagingPath, previous); err != nil {
					return 0, "", err
				}
				return previous, inUseBy, nil
			}
		}
	}

	if credUID >= 0 {
		// There is only one ticket cache per credential uid.
		if inUseBy, ok := principals[credUID]; ok && inUseBy != principal {
			return 0, "", fmt.Errorf("cruid %d is in use on this node by volumes mounted as another kerberos principal, leave out the cruid mount flag or use another one", credUID)
		} else if !ok && d.kerberosCacheExists(credUID) {
			return 0, "", fmt.Errorf("cruid %d is in use on this node by volumes mounted with a kerberos ticket cache, leave out the cruid mount flag or use another one", credUID)
		}
	} else {
		for uid, inUseBy := range principals {
			if inUseBy == principal {
				credUID = uid
				break
			}
		}
		if credUID < 0 {
			h := fnv.New32a()
			h.Write([]byte(principal))
			offset := int(h.Sum32() % krb5AllocatedUIDCount)
			for d.kerberosCacheExists(krb5AllocatedUIDBase+offset) || principals[krb5AllocatedUIDBase+offset] != "" {
				offset = (offset + 1) % krb5AllocatedUIDCount
			}
			credUID = krb5AllocatedUIDBase + offset
		}
	}

	if previous, ok := d.keytabVolumeCredUID(volumeFile); ok && previous != credUID {
		// the principal or the cruid of the volume, which is not mounted,
		// changed
		d.releaseKeytabCredential(volumeFile)
	}
	if _, ok := principals[credUID]; !ok {
		if err := writeFileAtomic(d.keytabFile(credUID, krb5PrincipalFileSuffix), []byte(principal+"\n")); err != nil {
			return 0, "", err
		}
	}
	if err := d.writeKeytabVolume(volumeID, stagingPath, credUID); err != nil {
		return 0, "", err
	}
	return credUID, principal, nil
}

// releaseKerberosKeytab records that a volume no longer uses its keytab, and
// removes the keytab, the principal and the ticket cache of its credential
// uid when no other volume uses them. It does nothing for a volume that is
// not staged with a keytab.
func (d *Driver) releaseKerberosKeytab(volumeID string) {
	d.krb5Mutex.Lock()
	defer d.krb5Mutex.Unlock()
	d.releaseKeytabCredential(d.keytabVolumeFile(volumeID))
}

// releaseKeytabCredential must be called with krb5Mutex held.
func (d *Driver) releaseKeytabCredential(volumeFile string) {
	credUID, ok := d.keytabVolumeCredUID(volumeFile)
	if !ok {
		return
	}
	if err := os.Remove(volumeFile); err != nil {
		klog.Errorf("failed to remove %s: %v", volumeFile, err)
		return
	}
	if d.keytabVolumeCount(credUID) > 0 {
		return
	}
	for _, file := range []string{
		d.keytabFile(credUID, krb5KeytabFileSuffix),
		d.keytabFile(credUID, krb5PrincipalFileSuffix),
		d.kerberosCacheFile(credUID),
	} {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			klog.Errorf("failed to remove %s: %v", file, err)
		}
	}
	klog.V(2).Infof("removed the kerberos keytab and ticket cache of cruid %d", credUID)
}

func tempFileSuffix() string {
	return fmt.Sprintf(".tmp.%d.%d", time.Now().UnixNano(), rand.Int63())
}

// writeFileAtomic writes a file readable by root only, replacing an existing
// file atomically.
func writeFileAtomic(path string, content []byte) error {
	temp := path + tempFileSuffix()
	if err := os.WriteFile(temp, content, 0600); err != nil {
		os.Remove(temp)
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		os.Remove(temp)
		return err
	}
	return nil
}

// refreshKeytabTicket gets a ticket for principal and makes it the ticket
// cache of credUID. With a keytab it is used to get the ticket, and it
// replaces the stored keytab of credUID when that succeeds, so that a keytab
// that does not work never replaces one that does. Without a keytab the
// stored one is used.
//
// kinit runs without holding krb5Mutex: it can take as long as kinitTimeout
// when the KDC does not answer.
func (d *Driver) refreshKeytabTicket(credUID int, principal string, keytab []byte) error {
	keytabPath := d.keytabFile(credUID, krb5KeytabFileSuffix)
	kinitKeytabPath := keytabPath
	if keytab != nil {
		kinitKeytabPath = keytabPath + tempFileSuffix()
		defer os.Remove(kinitKeytabPath)
		if err := os.WriteFile(kinitKeytabPath, keytab, 0600); err != nil {
			return err
		}
	}
	cachePath := d.kerberosCacheFile(credUID)
	tempCachePath := cachePath + tempFileSuffix()
	defer os.Remove(tempCachePath)

	ctx, cancel := context.WithTimeout(context.Background(), kinitTimeout)
	defer cancel()
	if err := kinitFunc(ctx, kinitKeytabPath, principal, tempCachePath); err != nil {
		return err
	}
	// cifs.upcall reads the cache as the cruid user
	if err := os.Chmod(tempCachePath, 0600); err != nil {
		return err
	}
	if err := chownFunc(tempCachePath, credUID, -1); err != nil {
		return fmt.Errorf("failed to chown kerberos cache %s to uid %d: %v", tempCachePath, credUID, err)
	}

	d.krb5Mutex.Lock()
	defer d.krb5Mutex.Unlock()
	if d.keytabPrincipals()[credUID] != principal {
		return errKeytabReleased
	}
	if keytab != nil {
		if err := os.Rename(kinitKeytabPath, keytabPath); err != nil {
			return err
		}
	}
	return os.Rename(tempCachePath, cachePath)
}

// ensureKerberosKeytab prepares the mount at stagingPath with the kerberos
// keytab of the secret: it gets a ticket for the principal and returns the
// mount flags with the cruid of the principal. usesKeytab is false, and
// nothing is done, when the mount is not a kerberos mount or the secret has
// no keytab. mounted tells whether the volume is already mounted at
// stagingPath.
//
// For a volume that is already mounted, a failure to get a ticket is not an
// error as long as there is a ticket cache: the volume works without a new
// ticket.
//
// When it returns an error, or when staging the volume fails afterwards, the
// caller must call releaseKerberosKeytab, unless the volume was already
// staged.
func (d *Driver) ensureKerberosKeytab(volumeID, stagingPath string, mounted bool, mountFlags []string, secrets map[string]string) (flags []string, usesKeytab bool, err error) {
	if !hasKerberosMountOption(mountFlags) {
		return mountFlags, false, nil
	}
	principal, keytab, found, err := getKerberosKeytab(secrets)
	if err != nil {
		return mountFlags, false, status.Error(codes.InvalidArgument, err.Error())
	}
	if !found {
		return mountFlags, false, nil
	}
	if _, err := kerberosCacheDirectoryExists(d.krb5CacheDirectory); err != nil {
		return mountFlags, false, err
	}
	if err := os.MkdirAll(d.keytabDirectory(), 0700); err != nil {
		return mountFlags, false, status.Errorf(codes.Internal, "failed to create %s: %v", d.keytabDirectory(), err)
	}

	credUID, credPrincipal, err := d.reserveKeytabCredential(volumeID, stagingPath, mounted, mountFlags, principal)
	if err != nil {
		return mountFlags, false, status.Error(codes.InvalidArgument, err.Error())
	}
	d.ensureKerberosPasswdEntry(credUID)
	// The keytab of the secret is not one of the principal of a mounted
	// volume whose principal changed: the volume keeps the stored keytab.
	if credPrincipal == principal {
		if err := d.refreshKeytabTicket(credUID, principal, keytab); err != nil {
			if !mounted || !d.kerberosCacheExists(credUID) {
				return mountFlags, false, status.Errorf(codes.Internal, "failed to get a kerberos ticket for %s with the keytab: %v", principal, err)
			}
			klog.Warningf("volume(%s) is already mounted, it keeps the kerberos ticket cache of %s (cruid %d): failed to get a new ticket with the keytab: %v", volumeID, principal, credUID, err)
		}
	}
	klog.V(2).Infof("volume(%s) uses the kerberos ticket of %s (cruid %d)", volumeID, credPrincipal, credUID)
	return withCredUID(mountFlags, credUID), true, nil
}

// withCredUID returns the mount flags with cruid=credUID, in place of the
// cruid mount options when there are any, also in a mount flag with several
// options.
func withCredUID(mountFlags []string, credUID int) []string {
	cruidOption := fmt.Sprintf("cruid=%d", credUID)
	flags := make([]string, 0, len(mountFlags)+1)
	found := false
	for _, mountFlag := range mountFlags {
		options := strings.Split(mountFlag, ",")
		changed := false
		for i, option := range options {
			if strings.HasPrefix(strings.TrimSpace(option), "cruid=") {
				options[i] = cruidOption
				changed = true
			}
		}
		if changed {
			mountFlag = strings.Join(options, ",")
			found = true
		}
		flags = append(flags, mountFlag)
	}
	if !found {
		flags = append(flags, cruidOption)
	}
	return flags
}

// pruneKeytabVolumes releases the keytab of volumes that are recorded as
// staged but whose staging path is no longer mounted. NodeUnstageVolume is not
// called for a volume that never got staged: when NodeStageVolume timed out
// and the mount failed afterwards, and kubelet did not try again (e.g. the pod
// was deleted). Volumes that are being staged or unstaged are skipped, and so
// are volumes staged by an older version of the driver, whose staging path is
// not known.
func (d *Driver) pruneKeytabVolumes() {
	files, _ := filepath.Glob(filepath.Join(d.keytabDirectory(), "*"+krb5VolumeFileSuffix))
	if len(files) == 0 {
		return
	}
	mountPoints, err := d.mounter.List()
	if err != nil {
		klog.Errorf("failed to list the mount points: %v", err)
		return
	}
	for _, file := range files {
		volume, ok := readKeytabVolume(file)
		if !ok || volume.VolumeID == "" || volume.StagingPath == "" || d.keytabVolumeFile(volume.VolumeID) != file {
			continue
		}
		// the lock of NodeStageVolume and NodeUnstageVolume
		lockKey := fmt.Sprintf("%s-%s", volume.VolumeID, volume.StagingPath)
		if !d.volumeLocks.TryAcquire(lockKey) {
			continue
		}
		if !d.isStagingPathMounted(volume.StagingPath, mountPoints) {
			klog.Warningf("volume(%s) is recorded as staged with a kerberos keytab, but %s is not mounted: releasing its keytab", volume.VolumeID, volume.StagingPath)
			d.releaseKerberosKeytab(volume.VolumeID)
		}
		d.volumeLocks.Release(lockKey)
	}
}

// isStagingPathMounted tells whether path may be mounted. Only a path that is
// neither a mount point nor in mountPoints is not mounted: mountPoints may
// have been listed before the volume was mounted. A path that can not be
// checked, e.g. a corrupted mount, counts as mounted.
func (d *Driver) isStagingPathMounted(path string, mountPoints []mount.MountPoint) bool {
	notMnt, err := d.mounter.IsLikelyNotMountPoint(path)
	if err != nil && !os.IsNotExist(err) {
		return true
	}
	if err == nil && !notMnt {
		return true
	}
	for _, mountPoint := range mountPoints {
		if mountPoint.Path == path || mountPoint.Path == filepath.Clean(path) {
			return true
		}
	}
	return false
}
