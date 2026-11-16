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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"k8s.io/klog/v2"
)

const defaultPasswdFile = "/etc/passwd"

// passwdMutex serializes updates of the passwd file.
var passwdMutex sync.Mutex

// ensurePasswdEntry makes sure passwdFile has an entry for uid, and appends
// one if it does not.
//
// cifs.upcall, which the kernel calls to get the kerberos ticket of the
// cruid user, exits with "Unable to find pw entry for uid" when the uid can
// not be resolved in the mount namespace of the process doing the mount (the
// smb container), and the mount fails with error 126 (required key not
// available). This is the case on platforms that run workloads with
// arbitrary uids, e.g. OpenShift, where the cruid is usually taken from the
// uid range of the namespace.
func ensurePasswdEntry(passwdFile string, uid int) error {
	if uid < 0 {
		return fmt.Errorf("invalid uid %d", uid)
	}
	passwdMutex.Lock()
	defer passwdMutex.Unlock()

	content, err := os.ReadFile(passwdFile)
	if err != nil {
		return err
	}
	uidStr := strconv.Itoa(uid)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) >= 3 && fields[2] == uidStr {
			return nil
		}
	}

	entry := fmt.Sprintf("csi-smb-krb5-%d:x:%d:%d:csi-driver-smb kerberos credential uid:/nonexistent:/usr/sbin/nologin\n", uid, uid, uid)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		entry = "\n" + entry
	}
	// Append in place instead of replacing the file, it may be a bind mount.
	f, err := os.OpenFile(passwdFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(entry); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	klog.V(2).Infof("added passwd entry for kerberos credential uid %d to %s", uid, passwdFile)
	return nil
}

// ensureKerberosPasswdEntry adds a passwd entry for credUID when
// --krb5-ensure-passwd-entry is set. A failure is not fatal: the uid may be
// resolvable by other means than the passwd file.
func (d *Driver) ensureKerberosPasswdEntry(credUID int) {
	if !d.krb5EnsurePasswdEntry {
		return
	}
	if err := ensurePasswdEntry(d.passwdFile, credUID); err != nil {
		klog.Warningf("failed to ensure a passwd entry for kerberos credential uid %d in %s: %v", credUID, d.passwdFile, err)
	}
}

// restoreKerberosPasswdEntries adds a passwd entry for the credential uid of
// every ticket cache in the kerberos cache directory (<krb5Prefix><uid>).
//
// The entries are added when a volume is staged, but they are part of the
// file system of the container: after a restart of the container they are
// gone while the volumes are still mounted, and cifs.upcall would fail when
// the kernel has to set up a new session for one of them.
func (d *Driver) restoreKerberosPasswdEntries() {
	if !d.krb5EnsurePasswdEntry {
		return
	}
	caches, err := filepath.Glob(filepath.Join(d.krb5CacheDirectory, d.krb5Prefix+"*"))
	if err != nil {
		klog.Errorf("failed to list kerberos ticket caches in %s: %v", d.krb5CacheDirectory, err)
		return
	}
	for _, cache := range caches {
		credUID, err := strconv.Atoi(strings.TrimPrefix(filepath.Base(cache), d.krb5Prefix))
		if err != nil || credUID < 0 {
			continue
		}
		d.ensureKerberosPasswdEntry(credUID)
	}
}
