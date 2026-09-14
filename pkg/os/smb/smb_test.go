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

func TestParseGlobalMappingAdditionalParams(t *testing.T) {
	tests := []struct {
		name                string
		requirePrivacy      bool
		globalMappingParams string
		expectedEnvs        []string
		expectErr           string
	}{
		{
			name:           "default require privacy true when no additional params provided",
			requirePrivacy: true,
			expectedEnvs:   []string{"smbopt_requireprivacy=true"},
		},
		{
			name:           "default require privacy false when no additional params provided",
			requirePrivacy: false,
			expectedEnvs:   []string{"smbopt_requireprivacy=false"},
		},
		{
			name:                "structured params are parsed into env vars",
			requirePrivacy:      true,
			globalMappingParams: "RequirePrivacy=false,RequireIntegrity=true,TransportType=QUIC,TcpPort=445,FullAccess=user1;user2",
			expectedEnvs: []string{
				"smbopt_requireprivacy=false",
				"smbopt_requireintegrity=true",
				"smbopt_transporttype=QUIC",
				"smbopt_tcpport=445",
				"smbopt_fullaccess=user1;user2",
			},
		},
		{
			name:                "reject invalid format",
			globalMappingParams: "RequirePrivacy false",
			expectErr:           "expected key=value",
		},
		{
			name:                "reject unsupported key",
			globalMappingParams: "WhatIf=true",
			expectErr:           "unsupported global mapping additional param",
		},
		{
			name:                "reject invalid boolean",
			globalMappingParams: "RequirePrivacy=maybe",
			expectErr:           "invalid boolean value",
		},
		{
			name:                "reject duplicate key",
			globalMappingParams: "RequirePrivacy=true,RequirePrivacy=false",
			expectErr:           "duplicate global mapping additional param",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			envs, err := parseGlobalMappingAdditionalParams(test.requirePrivacy, test.globalMappingParams)
			if test.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectErr) {
					t.Fatalf("expected error containing %q, got %v", test.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			for _, want := range test.expectedEnvs {
				if !containsString(envs, want) {
					t.Fatalf("expected envs to contain %q, got %v", want, envs)
				}
			}
		})
	}
}

func TestNewSmbGlobalMappingCmd(t *testing.T) {
	cmd := newSmbGlobalMappingCmd()
	for _, want := range []string{
		"New-SmbGlobalMapping @Params",
		"function HasValue([string]$Value) { return -not [string]::IsNullOrEmpty($Value) }",
		"if (HasValue $Env:smbopt_requireprivacy)",
		"$Env:smbuser",
		"$Env:smbpassword",
		"$Env:smbremotepath",
		"$Env:smbopt_requireprivacy",
		"$Env:smbopt_requireintegrity",
		"$Env:smbopt_tcpport",
		"$Env:smbopt_fullaccess",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("expected command to contain %q, got %q", want, cmd)
		}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
