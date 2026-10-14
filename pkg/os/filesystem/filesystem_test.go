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

package filesystem

import (
	"context"
	"strings"
	"testing"
)

func TestPathValidNonexistent(t *testing.T) {
	// Matches csi-proxy utils.IsPathValid: missing path returns (false, nil), not a PowerShell error.
	valid, err := PathValid(context.Background(), `Z:\csi-driver-smb-pathvalid-does-not-exist`)
	if err != nil {
		t.Fatalf("PathValid returned unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected nonexistent path to be invalid")
	}
}

func TestPathValidInvalidName(t *testing.T) {
	valid, err := PathValid(context.Background(), string([]byte{0x00}))
	if valid {
		t.Fatal("expected invalid path to be false")
	}
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("expected invalid path error, got %q", err.Error())
	}
}
