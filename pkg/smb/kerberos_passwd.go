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
// available). This is so with upcall_target=app (the default) and
// cifs-utils 7.6 or older on the host; see kerberos_userdb.go for the other
// cases. This is the case on platforms that run workloads with
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

	entry := fmt.Sprintf("%s:x:%d:%d:csi-driver-smb kerberos credential uid:/nonexistent:/usr/sbin/nologin\n", kerberosUserName(uid), uid, uid)
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
