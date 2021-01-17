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
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/kubernetes-csi/csi-driver-smb/test/utils/testutil"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

func matchFlakyWindowsError(mainError error, substr string) bool {
	var errorMessage string
	if mainError == nil {
		errorMessage = ""
	} else {
		errorMessage = mainError.Error()
	}

	return strings.Contains(errorMessage, substr)
}

func TestNodeStageVolume(t *testing.T) {
	stdVolCap := csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{},
		},
	}

	errorMountSensSource := testutil.GetWorkDirPath("error_mount_sens_source", t)
	smbFile := testutil.GetWorkDirPath("smb.go", t)
	sourceTest := testutil.GetWorkDirPath("source_test", t)

	volContext := map[string]string{
		sourceField: "test_source",
	}
	secrets := map[string]string{
		usernameField: "test_username",
		passwordField: "test_password",
		domainField:   "test_doamin",
	}

	tests := []struct {
		desc        string
		setup       func(*Driver)
		req         csi.NodeStageVolumeRequest
		expectedErr testutil.TestError
		cleanup     func(*Driver)

		// use this field only when Windows
		// gives flaky error messages due
		// to CSI proxy
		// This field holds the base error message
		// that is common amongst all other flaky
		// error messages
		flakyWindowsErrorMessage string
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  csi.NodeStageVolumeRequest{},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Volume capabilities missing",
			req:  csi.NodeStageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume capability not provided"),
			},
		},
		{
			desc: "[Error] Stage target path missing",
			req:  csi.NodeStageVolumeRequest{VolumeId: "vol_1", VolumeCapability: &stdVolCap},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Source field is missing in context",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "source field is missing, current context: map[]"),
			},
		},
		{
			desc: "[Error] Not a Directory",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: smbFile,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Internal, fmt.Sprintf("MkdirAll %s failed with error: mkdir %s: not a directory", smbFile, smbFile)),
				WindowsError: status.Error(codes.Internal, fmt.Sprintf("Could not mount target %s: mkdir %s: The system cannot find the path specified.", smbFile, smbFile)),
			},
		},
		{
			desc: "[Error] Volume operation in progress",
			setup: func(d *Driver) {
				d.volumeLocks.TryAcquire("vol_1")
			},
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release("vol_1")
			},
		},
		{
			desc: "[Error] Failed SMB mount mocked by MountSensitive",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: errorMountSensSource,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			flakyWindowsErrorMessage: fmt.Sprintf("volume(vol_1##) mount \"test_source\" on %#v failed "+
				"with smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				errorMountSensSource),
			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal,
					fmt.Sprintf("volume(vol_1##) mount \"test_source\" on \"%s\" failed with fake "+
						"MountSensitive: target error",
						errorMountSensSource)),
			},
		},
		{
			desc: "[Success] Valid request",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			flakyWindowsErrorMessage: fmt.Sprintf("volume(vol_1##) mount \"test_source\" on %#v failed with "+
				"smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				sourceTest),
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	d := NewFakeDriver()

	for _, test := range tests {
		mounter, err := NewFakeMounter()
		if err != nil {
			t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
		}
		d.mounter = mounter

		if test.setup != nil {
			test.setup(d)
		}
		_, err = d.NodeStageVolume(context.Background(), &test.req)

		// separate assertion for flaky error messages
		if test.flakyWindowsErrorMessage != "" && runtime.GOOS == "windows" {
			if !matchFlakyWindowsError(err, test.flakyWindowsErrorMessage) {
				t.Errorf("test case: %s, \nUnexpected error: %v\nExpected error: %v", test.desc, err, test.flakyWindowsErrorMessage)
			}
		} else {
			if !testutil.AssertError(&test.expectedErr, err) {
				t.Errorf("test case: %s, \nUnexpected error: %v\nExpected error: %v", test.desc, err, test.expectedErr.GetExpectedError())
			}
		}
		if test.cleanup != nil {
			test.cleanup(d)
		}
	}

	// Clean up
	err := os.RemoveAll(sourceTest)
	assert.NoError(t, err)
	err = os.RemoveAll(errorMountSensSource)
	assert.NoError(t, err)

}

func TestNodeGetInfo(t *testing.T) {
	d := NewFakeDriver()

	// Test valid request
	req := csi.NodeGetInfoRequest{}
	resp, err := d.NodeGetInfo(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, resp.GetNodeId(), fakeNodeID)
}

func TestNodeGetCapabilities(t *testing.T) {
	d := NewFakeDriver()
	capType := &csi.NodeServiceCapability_Rpc{
		Rpc: &csi.NodeServiceCapability_RPC{
			Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		},
	}
	capList := []*csi.NodeServiceCapability{{
		Type: capType,
	}}
	d.NSCap = capList
	// Test valid request
	req := csi.NodeGetCapabilitiesRequest{}
	resp, err := d.NodeGetCapabilities(context.Background(), &req)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.Capabilities[0].GetType(), capType)
	assert.NoError(t, err)
}

func TestNodeExpandVolume(t *testing.T) {
	d := NewFakeDriver()
	req := csi.NodeExpandVolumeRequest{}
	resp, err := d.NodeExpandVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNodePublishVolume(t *testing.T) {
	volumeCap := csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER}
	errorMountSource := testutil.GetWorkDirPath("error_mount_source", t)
	alreadyMountedTarget := testutil.GetWorkDirPath("false_is_likely_exist_target", t)
	smbFile := testutil.GetWorkDirPath("smb.go", t)
	sourceTest := testutil.GetWorkDirPath("source_test", t)
	targetTest := testutil.GetWorkDirPath("target_test", t)

	tests := []struct {
		desc          string
		setup         func(*Driver)
		req           csi.NodePublishVolumeRequest
		skipOnWindows bool
		expectedErr   testutil.TestError
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume capabilities missing",
			req:  csi.NodePublishVolumeRequest{},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume capability missing in request"),
			},
		},
		{
			desc: "[Error] Volume ID missing",
			req:  csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap}},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target path missing",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Target path not provided"),
			},
		},
		{
			desc: "[Error] Stage target path missing",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:   "vol_1",
				TargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Not a directory",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        smbFile,
				StagingTargetPath: sourceTest,
				Readonly:          true},

			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal, "Could not mount target \"%s\": mkdir %s: not a directory", smbFile, smbFile),
				WindowsError: status.Errorf(codes.Internal, "Could not mount target %#v: mkdir %s: The system cannot find the path specified.", smbFile, smbFile),
			},
		},
		{
			desc: "[Error] Volume operation in progress",
			setup: func(d *Driver) {
				d.volumeLocks.TryAcquire("vol_1")
			},
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release("vol_1")
			},
		},
		{
			desc: "[Error] Mount error mocked by Mount",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: errorMountSource,
				Readonly:          true},
			// todo: This test does not return any error on windows
			// Once the issue is figured out, we'll remove this field
			skipOnWindows: true,
			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal, fmt.Sprintf("Could not mount \"%s\" at \"%s\": fake Mount: source error", errorMountSource, targetTest)),
			},
		},
		{
			desc: "[Success] Valid request read only",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request already mounted",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        alreadyMountedTarget,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	_ = makeDir(alreadyMountedTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodePublishVolume(context.Background(), &test.req)
			if !testutil.AssertError(&test.expectedErr, err) {
				t.Errorf("test case: %s, \nUnexpected error: %v\nExpected error: %v", test.desc, err, test.expectedErr.GetExpectedError())
			}
			if test.cleanup != nil {
				test.cleanup(d)
			}
		}
	}

	// Clean up
	err = os.RemoveAll(targetTest)
	assert.NoError(t, err)
	err = os.RemoveAll(alreadyMountedTarget)
	assert.NoError(t, err)
}

func TestNodeUnpublishVolume(t *testing.T) {
	errorTarget := testutil.GetWorkDirPath("error_is_likely_target", t)
	targetFile := testutil.GetWorkDirPath("abc.go", t)
	targetTest := testutil.GetWorkDirPath("target_test", t)

	tests := []struct {
		desc          string
		setup         func(*Driver)
		req           csi.NodeUnpublishVolumeRequest
		expectedErr   testutil.TestError
		skipOnWindows bool
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  csi.NodeUnpublishVolumeRequest{TargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target missing",
			req:  csi.NodeUnpublishVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Target path missing in request"),
			},
		},
		{
			desc: "[Error] Volume operation in progress",
			setup: func(d *Driver) {
				d.volumeLocks.TryAcquire("vol_1")
			},
			req: csi.NodeUnpublishVolumeRequest{TargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release("vol_1")
			},
		},
		{
			desc: "[Error] Unmount error mocked by IsLikelyNotMountPoint",
			req:  csi.NodeUnpublishVolumeRequest{TargetPath: errorTarget, VolumeId: "vol_1"},
			// todo: This test does not return any error on windows
			// Once the issue is figured out, we'll remove this field
			skipOnWindows: true,
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Internal, fmt.Sprintf("failed to unmount target \"%s\": fake IsLikelyNotMountPoint: fake error", errorTarget)),
			},
		},
		{
			desc:        "[Success] Valid request",
			req:         csi.NodeUnpublishVolumeRequest{TargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodeUnpublishVolume(context.Background(), &test.req)
			if !testutil.AssertError(&test.expectedErr, err) {
				t.Errorf("test case: %s, \nUnexpected error: %v\nExpected error: %v", test.desc, err, test.expectedErr.GetExpectedError())
			}
			if test.cleanup != nil {
				test.cleanup(d)
			}
		}
	}

	// Clean up
	err = os.RemoveAll(errorTarget)
	assert.NoError(t, err)
}

func TestNodeUnstageVolume(t *testing.T) {
	errorTarget := testutil.GetWorkDirPath("error_is_likely_target", t)
	targetFile := testutil.GetWorkDirPath("abc.go", t)
	targetTest := testutil.GetWorkDirPath("target_test", t)

	tests := []struct {
		desc          string
		setup         func(*Driver)
		req           csi.NodeUnstageVolumeRequest
		skipOnWindows bool
		expectedErr   testutil.TestError
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  csi.NodeUnstageVolumeRequest{StagingTargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target missing",
			req:  csi.NodeUnstageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Volume operation in progress",
			setup: func(d *Driver) {
				d.volumeLocks.TryAcquire("vol_1")
			},
			req: csi.NodeUnstageVolumeRequest{StagingTargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release("vol_1")
			},
		},
		{
			desc:          "[Error] CleanupMountPoint error mocked by IsLikelyNotMountPoint",
			req:           csi.NodeUnstageVolumeRequest{StagingTargetPath: errorTarget, VolumeId: "vol_1"},
			skipOnWindows: true,
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Internal, fmt.Sprintf("failed to unmount staging target %#v: fake IsLikelyNotMountPoint: fake error", errorTarget)),
			},
		},
		{
			desc:        "[Success] Valid request",
			req:         csi.NodeUnstageVolumeRequest{StagingTargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodeUnstageVolume(context.Background(), &test.req)
			if !testutil.AssertError(&test.expectedErr, err) {
				t.Errorf("test case: %s, \nUnexpected error: %v\nExpected error: %v", test.desc, err, test.expectedErr.GetExpectedError())
			}
			if test.cleanup != nil {
				test.cleanup(d)
			}
		}
	}

	// Clean up
	err = os.RemoveAll(errorTarget)
	assert.NoError(t, err)
}

func TestEnsureMountPoint(t *testing.T) {
	errorTarget := "./error_is_likely_target"
	alreadyExistTarget := "./false_is_likely_exist_target"
	falseTarget := "./false_is_likely_target"
	smbFile := "./smb.go"
	targetTest := "./target_test"

	tests := []struct {
		desc        string
		target      string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked by IsLikelyNotMountPoint",
			target:      errorTarget,
			expectedErr: fmt.Errorf("fake IsLikelyNotMountPoint: fake error"),
		},
		{
			desc:        "[Error] Error opening file",
			target:      falseTarget,
			expectedErr: &os.PathError{Op: "open", Path: "./false_is_likely_target", Err: syscall.ENOENT},
		},
		{
			desc:        "[Error] Not a directory",
			target:      smbFile,
			expectedErr: &os.PathError{Op: "mkdir", Path: "./smb.go", Err: syscall.ENOTDIR},
		},
		{
			desc:        "[Success] Successful run",
			target:      targetTest,
			expectedErr: nil,
		},
		{
			desc:        "[Success] Already existing mount",
			target:      alreadyExistTarget,
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(alreadyExistTarget)
	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}

	for _, test := range tests {
		_, err := d.ensureMountPoint(test.target)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("test case: %s, Unexpected error: %v", test.desc, err)
		}
	}

	// Clean up
	err := os.RemoveAll(alreadyExistTarget)
	assert.NoError(t, err)
	err = os.RemoveAll(targetTest)
	assert.NoError(t, err)
}

func TestMakeDir(t *testing.T) {
	targetTest := "./target_test"

	//Successfully create directory
	err := makeDir(targetTest)
	assert.NoError(t, err)

	//Failed case
	err = makeDir("./smb.go")
	var e *os.PathError
	if !errors.As(err, &e) {
		t.Errorf("Unexpected Error: %v", err)
	}

	// Remove the directory created
	err = os.RemoveAll(targetTest)
	assert.NoError(t, err)
}

func TestNodeGetVolumeStats(t *testing.T) {
	nonexistedPath := "/not/a/real/directory"
	fakePath := "/tmp/fake-volume-path"

	tests := []struct {
		desc        string
		req         csi.NodeGetVolumeStatsRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume ID missing",
			req:         csi.NodeGetVolumeStatsRequest{VolumePath: fakePath},
			expectedErr: status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume ID was empty"),
		},
		{
			desc:        "[Error] VolumePath missing",
			req:         csi.NodeGetVolumeStatsRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume path was empty"),
		},
		{
			desc:        "[Error] Incorrect volume path",
			req:         csi.NodeGetVolumeStatsRequest{VolumePath: nonexistedPath, VolumeId: "vol_1"},
			expectedErr: status.Errorf(codes.NotFound, "path /not/a/real/directory does not exist"),
		},
		{
			desc:        "[Success] Standard success",
			req:         csi.NodeGetVolumeStatsRequest{VolumePath: fakePath, VolumeId: "vol_1"},
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(fakePath)
	d := NewFakeDriver()
	for _, test := range tests {
		_, err := d.NodeGetVolumeStats(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("desc: %v, expected error: %v, actual error: %v", test.desc, test.expectedErr, err)
		}
	}

	// Clean up
	err := os.RemoveAll(fakePath)
	assert.NoError(t, err)
}
