/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
)

// slowMounter wraps fakeMounter and blocks MountSensitive for a configurable
// duration. Used by TestMountWithTimeout to exercise timeout / lock semantics.
type slowMounter struct {
	fakeMounter
	delay time.Duration
}

func (s *slowMounter) MountSensitive(source, target, _ string, _ []string, _ []string) error {
	time.Sleep(s.delay)
	return s.fakeMounter.MountSensitive(source, target, "", nil, nil)
}

func TestMountWithTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows Mount() requires a real CSIProxyMounter; the timeout logic
		// under test is platform-agnostic and covered on Linux.
		t.Skip("skipping on windows: Mount() requires csi-proxy mounter")
	}
	tests := []struct {
		name           string
		mountDelay     time.Duration
		timeout        time.Duration
		wantKeepLock   bool
		wantCode       codes.Code
		wantErr        bool
		checkLockAsync bool // verify lock is released asynchronously
	}{
		{
			name:         "mount completes before timeout",
			mountDelay:   0,
			timeout:      time.Second,
			wantKeepLock: false,
			wantErr:      false,
		},
		{
			name:           "mount times out, lock held and released async",
			mountDelay:     500 * time.Millisecond,
			timeout:        50 * time.Millisecond,
			wantKeepLock:   true,
			wantCode:       codes.DeadlineExceeded,
			wantErr:        true,
			checkLockAsync: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewFakeDriver()
			d.mounter = &mount.SafeFormatAndMount{
				Interface: &slowMounter{delay: tc.mountDelay},
			}

			lockKey := "test-lock"
			if !d.volumeLocks.TryAcquire(lockKey) {
				t.Fatal("failed to acquire volume lock")
			}

			ctx := context.Background()
			keepLock, err := d.mountWithTimeout(ctx, "source", "/target", nil, nil, "vol-1", lockKey, tc.timeout)

			if keepLock != tc.wantKeepLock {
				t.Errorf("keepLockHeld = %v, want %v", keepLock, tc.wantKeepLock)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %v", err)
				}
				if st.Code() != tc.wantCode {
					t.Errorf("error code = %v, want %v", st.Code(), tc.wantCode)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tc.checkLockAsync {
				// Lock should be held right now (retry should fail)
				if d.volumeLocks.TryAcquire(lockKey) {
					t.Error("expected lock to be held after timeout, but TryAcquire succeeded")
					d.volumeLocks.Release(lockKey)
				}
				// Poll for the async lock release with an overall deadline so
				// we wait just long enough on slow CI runners without flaking.
				deadline := time.Now().Add(tc.mountDelay + 5*time.Second)
				acquired := false
				for time.Now().Before(deadline) {
					if d.volumeLocks.TryAcquire(lockKey) {
						acquired = true
						d.volumeLocks.Release(lockKey)
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if !acquired {
					t.Error("expected lock to be released after mount goroutine finished")
				}
			} else if !keepLock {
				// When keepLock is false, caller releases the lock — just clean up
				d.volumeLocks.Release(lockKey)
			}
		})
	}
}

func TestMount(t *testing.T) {
	targetTest := "./target_test"
	sourceTest := "./source_test"

	tests := []struct {
		desc        string
		source      string
		target      string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked source error",
			source:      "./error_mount_source",
			target:      targetTest,
			expectedErr: fmt.Errorf("fake Mount: source error"),
		},
		{
			desc:        "[Error] Mocked target error",
			source:      sourceTest,
			target:      "./error_mount_target",
			expectedErr: fmt.Errorf("fake Mount: target error"),
		},
		{
			desc:        "[Success] Successful run",
			source:      sourceTest,
			target:      targetTest,
			expectedErr: nil,
		},
	}

	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}
	for _, test := range tests {
		err := d.mounter.Mount(test.source, test.target, "", nil)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestMountSensitive(t *testing.T) {
	targetTest := "./target_test"
	sourceTest := "./source_test"

	tests := []struct {
		desc        string
		source      string
		target      string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked source error",
			source:      "./error_mount_sens_source",
			target:      targetTest,
			expectedErr: fmt.Errorf("fake MountSensitive: source error"),
		},
		{
			desc:        "[Error] Mocked target error",
			source:      sourceTest,
			target:      "./error_mount_sens_target",
			expectedErr: fmt.Errorf("fake MountSensitive: target error"),
		},
		{
			desc:        "[Success] Successful run",
			source:      sourceTest,
			target:      targetTest,
			expectedErr: nil,
		},
	}

	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}
	for _, test := range tests {
		err := d.mounter.MountSensitive(test.source, test.target, "", nil, nil)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestIsLikelyNotMountPoint(t *testing.T) {
	targetTest := "./target_test"
	tests := []struct {
		desc        string
		file        string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked file error",
			file:        "./error_is_likely_target",
			expectedErr: fmt.Errorf("fake IsLikelyNotMountPoint: fake error"),
		},
		{desc: "[Success] Successful run",
			file:        targetTest,
			expectedErr: nil,
		},
		{
			desc:        "[Success] Successful run not a mount",
			file:        "./false_is_likely_target",
			expectedErr: nil,
		},
	}

	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}
	for _, test := range tests {
		_, err := d.mounter.IsLikelyNotMountPoint(test.file)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}
