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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEnsurePasswdEntry(t *testing.T) {
	base := "root:x:0:0:root:/root:/bin/bash\nnobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin"
	tests := []struct {
		desc      string
		uid       int
		wantAdded bool
		wantErr   bool
	}{
		{desc: "existing uid is left alone", uid: 65534},
		{desc: "root is left alone", uid: 0},
		{desc: "missing uid is added", uid: 1001070000, wantAdded: true},
		{desc: "negative uid is refused", uid: -1, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			usersFile := filepath.Join(t.TempDir(), "passwd")
			if err := os.WriteFile(usersFile, []byte(base), 0644); err != nil {
				t.Fatal(err)
			}
			// twice: the second call must not add the entry again
			for i := 0; i < 2; i++ {
				err := ensurePasswdEntry(usersFile, test.uid)
				if (err != nil) != test.wantErr {
					t.Fatalf("ensurePasswdEntry: err = %v, wantErr = %t", err, test.wantErr)
				}
			}
			content, err := os.ReadFile(usersFile)
			if err != nil {
				t.Fatal(err)
			}
			if !test.wantAdded {
				if string(content) != base {
					t.Errorf("passwd file changed: %q", content)
				}
				return
			}
			want := base + "\ncsi-smb-krb5-1001070000:x:1001070000:1001070000:csi-driver-smb kerberos credential uid:/nonexistent:/usr/sbin/nologin\n"
			if string(content) != want {
				t.Errorf("passwd file = %q, want %q", content, want)
			}
		})
	}

	if err := ensurePasswdEntry(filepath.Join(t.TempDir(), "missing"), 1000); err == nil {
		t.Error("expected an error for a missing passwd file")
	}
}

func TestRestoreKerberosUserEntriesPasswd(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("kerberos is only supported on Linux")
	}
	cacheDir := t.TempDir()
	usersFile := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(usersFile, []byte("root:x:0:0:root:/root:/bin/bash\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// a ticket cache, the cache file of a volume and a file that is not the cache of a uid
	for _, name := range []string{"krb5cc_1001070000", "volume1", "krb5cc_notanumber"} {
		if err := os.WriteFile(filepath.Join(cacheDir, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// a ticket cache that is a symlink to the cache file of a volume
	if err := os.Symlink(filepath.Join(cacheDir, "volume1"), filepath.Join(cacheDir, "krb5cc_1000")); err != nil {
		t.Fatal(err)
	}

	d := NewFakeDriver()
	d.krb5CacheDirectory = cacheDir
	d.usersFile = usersFile

	d.krb5EnsurePasswdEntry = false
	d.restoreKerberosUserEntries()
	if content, _ := os.ReadFile(usersFile); strings.Count(string(content), "\n") != 1 {
		t.Errorf("passwd file changed although krb5EnsurePasswdEntry is off: %q", content)
	}

	d.krb5EnsurePasswdEntry = true
	d.restoreKerberosUserEntries()
	d.restoreKerberosUserEntries()
	content, _ := os.ReadFile(usersFile)
	for _, uid := range []string{"1001070000", "1000"} {
		if strings.Count(string(content), ":x:"+uid+":"+uid+":") != 1 {
			t.Errorf("expected exactly one passwd entry for uid %s, got: %q", uid, content)
		}
	}
	if strings.Count(string(content), "\n") != 3 {
		t.Errorf("unexpected passwd entries: %q", content)
	}
}
