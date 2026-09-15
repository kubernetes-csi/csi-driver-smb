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
	"fmt"
	"strings"
	"testing"
)

func TestParseSMBGlobalMappingStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want SMBGlobalMappingStatus
	}{
		{name: "ok", in: "OK", want: SMBGlobalMappingStatusOK},
		{name: "disconnected", in: "Disconnected", want: SMBGlobalMappingStatusDisconnected},
		{name: "notfound explicit", in: "NotFound", want: SMBGlobalMappingStatusNotFound},
		{name: "empty means notfound", in: "", want: SMBGlobalMappingStatusNotFound},
		{name: "other state", in: "Reconnecting", want: SMBGlobalMappingStatusOther},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseSMBGlobalMappingStatus(test.in); got != test.want {
				t.Fatalf("parseSMBGlobalMappingStatus(%q) = %q, want %q", test.in, got, test.want)
			}
		})
	}
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

func TestNewSmbGlobalMappingCmd(t *testing.T) {
	tests := []struct {
		name           string
		requirePrivacy bool
		expectedFlag   string
		unexpectedFlag string
	}{
		{
			name:           "require privacy true emits $true",
			requirePrivacy: true,
			expectedFlag:   "-RequirePrivacy $true",
			unexpectedFlag: "-RequirePrivacy $false",
		},
		{
			name:           "require privacy false emits $false",
			requirePrivacy: false,
			expectedFlag:   "-RequirePrivacy $false",
			unexpectedFlag: "-RequirePrivacy $true",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newSmbGlobalMappingCmd(test.requirePrivacy)
			if !strings.Contains(cmd, test.expectedFlag) {
				t.Errorf("expected command to contain %q, got %q", test.expectedFlag, cmd)
			}
			if strings.Contains(cmd, test.unexpectedFlag) {
				t.Errorf("expected command NOT to contain %q, got %q", test.unexpectedFlag, cmd)
			}
			// sanity: the New-SmbGlobalMapping invocation and env-var substitution must remain intact
			for _, want := range []string{
				"New-SmbGlobalMapping",
				"$Env:smbuser",
				"$Env:smbpassword",
				"$Env:smbremotepath",
			} {
				if !strings.Contains(cmd, want) {
					t.Errorf("expected command to contain %q, got %q", want, cmd)
				}
			}
		})
	}
}
