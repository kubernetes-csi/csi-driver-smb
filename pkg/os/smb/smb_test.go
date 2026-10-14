//go:build windows
// +build windows

/*
Copyright 2024 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"testing"

	"github.com/kubernetes-csi/csi-proxy/v2/pkg/wmi"
)

func TestCanonicalizeSMBRemotePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "normalizes separators case and trailing slash", in: `//SERVER/Share/`, want: `\\server\share`},
		{name: "preserves normalized unc path", in: `\\server\share`, want: `\\server\share`},
		{name: "strips repeated trailing backslashes", in: `\\SERVER\Share\\\\`, want: `\\server\share`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanonicalizeSMBRemotePath(test.in); got != test.want {
				t.Fatalf("CanonicalizeSMBRemotePath(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
}

func TestSMBGlobalMappingStatusFromWMI(t *testing.T) {
	tests := []struct {
		name   string
		status uint32
		want   SMBGlobalMappingStatus
	}{
		{name: "ok", status: wmi.SmbMappingStatusOK, want: SMBGlobalMappingStatusOK},
		{name: "disconnected", status: wmi.SmbMappingStatusDisconnected, want: SMBGlobalMappingStatusDisconnected},
		{name: "other", status: wmi.SmbMappingStatusReconnecting, want: SMBGlobalMappingStatusOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := smbGlobalMappingStatusFromWMI(test.status); got != test.want {
				t.Fatalf("smbGlobalMappingStatusFromWMI(%d) = %q, want %q", test.status, got, test.want)
			}
		})
	}
}

func TestGetSmbGlobalMappingStatus(t *testing.T) {
	originalQuery := querySmbGlobalMappingWMI
	originalGetStatus := getSmbGlobalMappingStatusWMI
	t.Cleanup(func() {
		querySmbGlobalMappingWMI = originalQuery
		getSmbGlobalMappingStatusWMI = originalGetStatus
	})

	t.Run("found and canonicalizes path", func(t *testing.T) {
		var gotPath string
		querySmbGlobalMappingWMI = func(_ *wmi.Scope, remotePath string) (*wmi.COMDispatchObject, error) {
			gotPath = remotePath
			return nil, nil
		}
		getSmbGlobalMappingStatusWMI = func(_ *wmi.COMDispatchObject) (uint32, error) {
			return wmi.SmbMappingStatusOK, nil
		}

		got, err := GetSmbGlobalMappingStatus(`//SERVER/Share/`)
		if err != nil {
			t.Fatalf("GetSmbGlobalMappingStatus returned error: %v", err)
		}
		if got != SMBGlobalMappingStatusOK {
			t.Fatalf("status = %q, want %q", got, SMBGlobalMappingStatusOK)
		}
		if gotPath != `\\server\share` {
			t.Fatalf("query path = %q, want %q", gotPath, `\\server\share`)
		}
	})

	t.Run("not found is reported without error", func(t *testing.T) {
		querySmbGlobalMappingWMI = func(_ *wmi.Scope, _ string) (*wmi.COMDispatchObject, error) {
			return nil, wmi.ErrNotFound
		}

		got, err := GetSmbGlobalMappingStatus(`\\server\share`)
		if err != nil {
			t.Fatalf("GetSmbGlobalMappingStatus returned error: %v", err)
		}
		if got != SMBGlobalMappingStatusNotFound {
			t.Fatalf("status = %q, want %q", got, SMBGlobalMappingStatusNotFound)
		}
	})

	t.Run("property read errors are returned", func(t *testing.T) {
		querySmbGlobalMappingWMI = func(_ *wmi.Scope, _ string) (*wmi.COMDispatchObject, error) {
			return nil, nil
		}
		getSmbGlobalMappingStatusWMI = func(_ *wmi.COMDispatchObject) (uint32, error) {
			return 0, errors.New("status read failed")
		}

		_, err := GetSmbGlobalMappingStatus(`\\server\share`)
		if err == nil || err.Error() != "status read failed" {
			t.Fatalf("error = %v, want status read failed", err)
		}
	})
}

func TestCheckForDuplicateSMBMounts(t *testing.T) {
	tests := []struct {
		name           string
		dir            string
		mount          string
		remoteServer   string
		expectedResult bool
		expectedError  error
	}{
		{
			name:           "directory does not exist",
			dir:            "non-existing-mount",
			expectedResult: false,
			expectedError:  fmt.Errorf("open non-existing-mount: The system cannot find the file specified."),
		},
	}

	for _, test := range tests {
		result, err := CheckForDuplicateSMBMounts(test.dir, test.mount, test.remoteServer)
		if result != test.expectedResult {
			t.Errorf("Expected %v, got %v", test.expectedResult, result)
		}
		if err == nil && test.expectedError != nil {
			t.Errorf("Expected error %v, got nil", test.expectedError)
		}
		if err != nil && test.expectedError == nil {
			t.Errorf("Expected nil, got %v", err)
		}
		if err != nil && test.expectedError != nil && err.Error() != test.expectedError.Error() {
			t.Errorf("Expected error %v, got %v", test.expectedError, err)
		}
	}
}

func TestNewSmbGlobalMappingPassesRequirePrivacy(t *testing.T) {
	tests := []struct {
		name           string
		requirePrivacy bool
	}{
		{name: "require privacy true", requirePrivacy: true},
		{name: "require privacy false", requirePrivacy: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotPrivacy bool
			var called bool
			orig := newSmbGlobalMappingWMI
			t.Cleanup(func() { newSmbGlobalMappingWMI = orig })
			newSmbGlobalMappingWMI = func(remotePath, username, password string, requirePrivacy bool) error {
				called = true
				gotPrivacy = requirePrivacy
				if remotePath != `\\server\share` || username != "user" || password != "pass" {
					t.Fatalf("unexpected args: path=%q user=%q pass=%q", remotePath, username, password)
				}
				return nil
			}

			if err := NewSmbGlobalMapping(`\\server\share`, "user", "pass", test.requirePrivacy); err != nil {
				t.Fatalf("NewSmbGlobalMapping returned error: %v", err)
			}
			if !called {
				t.Fatal("expected WMI create helper to be called")
			}
			if gotPrivacy != test.requirePrivacy {
				t.Fatalf("RequirePrivacy: got %v, want %v", gotPrivacy, test.requirePrivacy)
			}
		})
	}
}
