//go:build windows
// +build windows

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

package mounter

import (
	"fmt"
	"testing"

	"github.com/kubernetes-csi/csi-driver-smb/pkg/os/smb"
)

func TestEnsureHostProcessSMBGlobalMapping(t *testing.T) {
	tests := []struct {
		name          string
		status        smb.SMBGlobalMappingStatus
		pathValid     bool
		pathValidErr  error
		statusErr     error
		removeErr     error
		newErr        error
		wantRemove    bool
		wantNew       bool
		wantPathValid bool
		wantErr       bool
	}{
		{
			name:          "ok and valid reuses mapping",
			status:        smb.SMBGlobalMappingStatusOK,
			pathValid:     true,
			wantPathValid: true,
		},
		{
			name:          "ok but invalid removes and recreates mapping",
			status:        smb.SMBGlobalMappingStatusOK,
			pathValid:     false,
			wantRemove:    true,
			wantNew:       true,
			wantPathValid: true,
		},
		{
			name:       "disconnected removes and recreates mapping",
			status:     smb.SMBGlobalMappingStatusDisconnected,
			wantRemove: true,
			wantNew:    true,
		},
		{
			name:       "other unhealthy state removes and recreates mapping",
			status:     smb.SMBGlobalMappingStatusOther,
			wantRemove: true,
			wantNew:    true,
		},
		{
			name:    "not found creates mapping",
			status:  smb.SMBGlobalMappingStatusNotFound,
			wantNew: true,
		},
		{
			name:    "status lookup error falls back to create",
			status:  smb.SMBGlobalMappingStatusNotFound,
			statusErr: fmt.Errorf("lookup failed"),
			wantNew: true,
		},
		{
			name:          "path validation error still recreates when invalid",
			status:        smb.SMBGlobalMappingStatusOK,
			pathValid:     false,
			pathValidErr:  fmt.Errorf("validation failed"),
			wantPathValid: true,
			wantRemove:    true,
			wantNew:       true,
		},
		{
			name:       "remove failure is returned",
			status:     smb.SMBGlobalMappingStatusDisconnected,
			removeErr:  fmt.Errorf("remove failed"),
			wantRemove: true,
			wantErr:    true,
		},
		{
			name:    "new mapping failure is returned",
			status:  smb.SMBGlobalMappingStatusNotFound,
			newErr:  fmt.Errorf("new failed"),
			wantNew: true,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotRemove, gotNew, gotPathValid bool
			err := ensureHostProcessSMBGlobalMapping(
				`\\server\share`,
				"user",
				"password",
				true,
				func(string) (smb.SMBGlobalMappingStatus, error) {
					return test.status, test.statusErr
				},
				func(string) (bool, error) {
					gotPathValid = true
					return test.pathValid, test.pathValidErr
				},
				func(string) error {
					gotRemove = true
					return test.removeErr
				},
				func(remotePath, username, password string, requirePrivacy bool) error {
					gotNew = true
					if remotePath != `\\server\share` || username != "user" || password != "password" || !requirePrivacy {
						t.Fatalf("unexpected NewSmbGlobalMapping args: %q %q %q %v", remotePath, username, password, requirePrivacy)
					}
					return test.newErr
				},
			)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr=%v", err, test.wantErr)
			}
			if gotRemove != test.wantRemove {
				t.Fatalf("remove called = %v, want %v", gotRemove, test.wantRemove)
			}
			if gotNew != test.wantNew {
				t.Fatalf("new called = %v, want %v", gotNew, test.wantNew)
			}
			if gotPathValid != test.wantPathValid {
				t.Fatalf("pathValid called = %v, want %v", gotPathValid, test.wantPathValid)
			}
		})
	}
}
