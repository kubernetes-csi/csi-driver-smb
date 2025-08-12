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
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/kubernetes-csi/csi-driver-smb/test/utils/testutil"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mount "k8s.io/mount-utils"
	"k8s.io/utils/exec"
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
	mountGroupVolCap := csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				VolumeMountGroup: "1000",
			},
		},
	}
	mountGroupWithModesVolCap := csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{
				VolumeMountGroup: "1000",
				MountFlags:       []string{"file_mode=0111", "dir_mode=0111"},
			},
		},
	}

	errorMountSensSource := testutil.GetWorkDirPath("error_mount_sens_source", t)
	smbFile := testutil.GetWorkDirPath("smb.go", t)
	sourceTest := testutil.GetWorkDirPath("source_test", t)

	testSource := "\\\\hostname\\share\\test"
	volContext := map[string]string{
		sourceField: testSource,
	}
	volContextWithMetadata := map[string]string{
		sourceField:     testSource,
		pvcNameKey:      "pvcname",
		pvcNamespaceKey: "pvcnamespace",
		pvNameKey:       "pvname",
	}
	secrets := map[string]string{
		usernameField: "test_username",
		passwordField: "test_password",
		domainField:   "test_doamin",
	}
	secretsSpecial := map[string]string{
		usernameField: "test_username",
		passwordField: "test\"`,password",
		domainField:   "test_doamin",
	}

	tests := []struct {
		desc        string
		setup       func(*Driver)
		req         *csi.NodeStageVolumeRequest
		expectedErr testutil.TestError
		cleanup     func(*Driver)

		// use this field only when Windows
		// gives flaky error messages due
		// to CSI proxy
		// This field holds the base error message
		// that is common amongst all other flaky
		// error messages
		flakyWindowsErrorMessage string
		skipOnWindows            bool
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  &csi.NodeStageVolumeRequest{},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Volume capabilities missing",
			req:  &csi.NodeStageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume capability not provided"),
			},
		},
		{
			desc: "[Error] Stage target path missing",
			req:  &csi.NodeStageVolumeRequest{VolumeId: "vol_1", VolumeCapability: &stdVolCap},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Source field is missing in context",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "source field is missing, current context: map[]"),
			},
		},
		{
			desc: "[Error] Not a Directory",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: smbFile,
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
				d.volumeLocks.TryAcquire(fmt.Sprintf("%s-%s", "vol_1", sourceTest))
			},
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release(fmt.Sprintf("%s-%s", "vol_1", sourceTest))
			},
		},
		{
			desc: "[Error] Failed SMB mount mocked by MountSensitive",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: errorMountSensSource,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed "+
				"with NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), errorMountSensSource, testSource, errorMountSensSource),
			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal,
					"volume(vol_1##) mount \"%s\" on \"%s\" failed with fake "+
						"MountSensitive: target error",
					strings.Replace(testSource, "\\", "\\\\", -1), errorMountSensSource),
			},
		},
		{
			desc: "[Error] Failed SMB mount mocked by MountSensitive (password with special characters)",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: errorMountSensSource,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secretsSpecial},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed "+
				"with NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), errorMountSensSource, testSource, errorMountSensSource),
			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal,
					"volume(vol_1##) mount \"%s\" on \"%s\" failed with fake "+
						"MountSensitive: target error",
					strings.Replace(testSource, "\\", "\\\\", -1), errorMountSensSource),
			},
		},
		{
			desc: "[Success] Valid request",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed with "+
				"NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), sourceTest, testSource, sourceTest),
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request with pv/pvc metadata",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContextWithMetadata,
				Secrets:          secrets},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed with "+
				"NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), sourceTest, testSource, sourceTest),
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request with VolumeMountGroup",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &mountGroupVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed with "+
				"NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), sourceTest, testSource, sourceTest),
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request with VolumeMountGroup and file/dir modes",
			req: &csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &mountGroupWithModesVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			skipOnWindows: true,
			flakyWindowsErrorMessage: fmt.Sprintf("rpc error: code = Internal desc = volume(vol_1##) mount \"%s\" on %#v failed with "+
				"NewSmbGlobalMapping(%s, %s) failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				strings.Replace(testSource, "\\", "\\\\", -1), sourceTest, testSource, sourceTest),
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	d := NewFakeDriver()

	for _, test := range tests {
		if test.skipOnWindows && runtime.GOOS == "windows" {
			continue
		}
		mounter, err := NewFakeMounter()
		if err != nil {
			t.Fatalf("failed to get fake mounter: %v", err)
		}
		d.mounter = mounter

		if test.setup != nil {
			test.setup(d)
		}
		_, err = d.NodeStageVolume(context.Background(), test.req)

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
		req           *csi.NodePublishVolumeRequest
		skipOnWindows bool
		expectedErr   testutil.TestError
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume capabilities missing",
			req:  &csi.NodePublishVolumeRequest{},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume capability missing in request"),
			},
		},
		{
			desc: "[Error] Volume ID missing",
			req:  &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap}},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target path missing",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Target path not provided"),
			},
		},
		{
			desc: "[Error] Stage target path missing",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:   "vol_1",
				TargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Not a directory",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
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
			desc: "[Error] Mount error mocked by Mount",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: errorMountSource,
				Readonly:          true},
			// todo: This test does not return any error on windows
			// Once the issue is figured out, we'll remove this field
			skipOnWindows: true,
			expectedErr: testutil.TestError{
				DefaultError: status.Errorf(codes.Internal, "Could not mount \"%s\" at \"%s\": fake Mount: source error", errorMountSource, targetTest),
			},
		},
		{
			desc: "[Success] Valid request read only",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request already mounted",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        alreadyMountedTarget,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Success] Valid request",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: testutil.TestError{},
		},
		{
			desc: "[Error] failed to create ephemeral Volume",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true,
				VolumeContext:     map[string]string{ephemeralField: "true"},
			},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "source field is missing, current context: map[csi.storage.k8s.io/ephemeral:true secretnamespace:]"),
			},
		},
		{
			desc: "[error] failed request with ephemeral Volume",
			req: &csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true,
				VolumeContext: map[string]string{
					ephemeralField:    "true",
					sourceField:       "source",
					podNamespaceField: "podnamespace",
				},
			},
			skipOnWindows: true,
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Internal, "Error getting username and password from secret  in namespace podnamespace: could not username and password from secret(): KubeClient is nil"),
			},
		},
	}

	// Setup
	_ = makeDir(alreadyMountedTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf("failed to get fake mounter: %v", err)
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodePublishVolume(context.Background(), test.req)
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
		req           *csi.NodeUnpublishVolumeRequest
		expectedErr   testutil.TestError
		skipOnWindows bool
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  &csi.NodeUnpublishVolumeRequest{TargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target missing",
			req:  &csi.NodeUnpublishVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Target path missing in request"),
			},
		},
		{
			desc:        "[Success] Valid request",
			req:         &csi.NodeUnpublishVolumeRequest{TargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf("failed to get fake mounter: %v", err)
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodeUnpublishVolume(context.Background(), test.req)
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
		req           *csi.NodeUnstageVolumeRequest
		skipOnWindows bool
		expectedErr   testutil.TestError
		cleanup       func(*Driver)
	}{
		{
			desc: "[Error] Volume ID missing",
			req:  &csi.NodeUnstageVolumeRequest{StagingTargetPath: targetTest},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
			},
		},
		{
			desc: "[Error] Target missing",
			req:  &csi.NodeUnstageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.InvalidArgument, "Staging target not provided"),
			},
		},
		{
			desc: "[Error] Volume operation in progress",
			setup: func(d *Driver) {
				d.volumeLocks.TryAcquire(fmt.Sprintf("%s-%s", "vol_1", targetFile))
			},
			req: &csi.NodeUnstageVolumeRequest{StagingTargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{
				DefaultError: status.Error(codes.Aborted, fmt.Sprintf(volumeOperationAlreadyExistsFmt, "vol_1")),
			},
			cleanup: func(d *Driver) {
				d.volumeLocks.Release(fmt.Sprintf("%s-%s", "vol_1", targetFile))
			},
		},
		{
			desc:        "[Success] Valid request",
			req:         &csi.NodeUnstageVolumeRequest{StagingTargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: testutil.TestError{},
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf("failed to get fake mounter: %v", err)
	}
	d.mounter = mounter

	for _, test := range tests {
		if !(test.skipOnWindows && runtime.GOOS == "windows") {
			if test.setup != nil {
				test.setup(d)
			}
			_, err := d.NodeUnstageVolume(context.Background(), test.req)
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
		req         *csi.NodeGetVolumeStatsRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume ID missing",
			req:         &csi.NodeGetVolumeStatsRequest{VolumePath: fakePath},
			expectedErr: status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume ID was empty"),
		},
		{
			desc:        "[Error] VolumePath missing",
			req:         &csi.NodeGetVolumeStatsRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume path was empty"),
		},
		{
			desc:        "[Error] Incorrect volume path",
			req:         &csi.NodeGetVolumeStatsRequest{VolumePath: nonexistedPath, VolumeId: "vol_1"},
			expectedErr: status.Errorf(codes.NotFound, "path /not/a/real/directory does not exist"),
		},
		{
			desc:        "[Success] Standard success",
			req:         &csi.NodeGetVolumeStatsRequest{VolumePath: fakePath, VolumeId: "vol_1"},
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(fakePath)
	d := NewFakeDriver()
	for _, test := range tests {
		_, err := d.NodeGetVolumeStats(context.Background(), test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("desc: %v, expected error: %v, actual error: %v", test.desc, test.expectedErr, err)
		}
	}

	// Clean up
	err := os.RemoveAll(fakePath)
	assert.NoError(t, err)
}

func TestCheckGidPresentInMountFlags(t *testing.T) {
	tests := []struct {
		desc       string
		MountFlags []string
		result     bool
	}{
		{
			desc:       "[Success] Gid present in mount flags",
			MountFlags: []string{"gid=3000"},
			result:     true,
		},
		{
			desc:       "[Success] Gid not present in mount flags",
			MountFlags: []string{},
			result:     false,
		},
	}

	for _, test := range tests {
		gIDPresent := checkGidPresentInMountFlags(test.MountFlags)
		if gIDPresent != test.result {
			t.Errorf("[%s]: Expected result : %t, Actual result: %t", test.desc, test.result, gIDPresent)
		}
	}
}

func TestVolumeKerberosCacheName(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "s", // short name
		},
		{
			name: "Volume Handle##unique suffix",
		},
		{
			name: "Volume With Spaces and Slashes // and symbols that produce /+ after base64 ???????~~~~~~~~",
		},
	}

	for _, test := range tests {
		fileName := volumeKerberosCacheName(test.name)
		if strings.Contains(fileName, "/") || strings.Contains(fileName, "+") {
			t.Errorf("[%s]: Expected result should not contain / or +, Actual result: %s", test.name, fileName)
		}
	}
}

func TestHasKerberosMountOption(t *testing.T) {
	tests := []struct {
		desc       string
		MountFlags []string
		result     bool
	}{
		{
			desc:       "[Success] Sec kerberos present in mount flags",
			MountFlags: []string{"sec=krb5"},
			result:     true,
		},
		{
			desc:       "[Success] Sec kerberos present in mount flags",
			MountFlags: []string{"sec=krb5i"},
			result:     true,
		},
		{
			desc:       "[Success] Sec kerberos not present in mount flags",
			MountFlags: []string{},
			result:     false,
		},
		{
			desc:       "[Success] Sec kerberos not present in mount flags",
			MountFlags: []string{"sec=ntlm"},
			result:     false,
		},
	}

	for _, test := range tests {
		securityIsKerberos := hasKerberosMountOption(test.MountFlags)
		if securityIsKerberos != test.result {
			t.Errorf("[%s]: Expected result : %t, Actual result: %t", test.desc, test.result, securityIsKerberos)
		}
	}
}

func TestGetCredUID(t *testing.T) {
	_, convertErr := strconv.Atoi("foo")
	tests := []struct {
		desc        string
		MountFlags  []string
		result      int
		expectedErr error
	}{
		{
			desc:        "[Success] Got correct credUID",
			MountFlags:  []string{"cruid=1000"},
			result:      1000,
			expectedErr: nil,
		},
		{
			desc:        "[Success] Got correct credUID",
			MountFlags:  []string{"cruid=0"},
			result:      0,
			expectedErr: nil,
		},
		{
			desc:        "[Error] Got error when no CredUID",
			MountFlags:  []string{},
			result:      -1,
			expectedErr: fmt.Errorf("Can't find credUid in mount flags"),
		},
		{
			desc:        "[Error] Got error when CredUID is not an int",
			MountFlags:  []string{"cruid=foo"},
			result:      0,
			expectedErr: convertErr,
		},
	}

	for _, test := range tests {
		credUID, err := getCredUID(test.MountFlags)
		if credUID != test.result {
			t.Errorf("[%s]: Expected result : %d, Actual result: %d", test.desc, test.result, credUID)
		}
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("[%s]: Expected error : %v, Actual error: %v", test.desc, test.expectedErr, err)
		}
	}
}

func TestGetKerberosCache(t *testing.T) {
	ticket := []byte{'G', 'O', 'L', 'A', 'N', 'G'}
	base64Ticket := base64.StdEncoding.EncodeToString(ticket)
	credUID := 1000
	krb5CacheDirectory := "/var/lib/kubelet/kerberos/"
	krb5Prefix := "krb5cc_"
	goodFileName := fmt.Sprintf("%s%s%d", krb5CacheDirectory, krb5Prefix, credUID)
	krb5CcacheName := "krb5cc_1000"

	_, base64DecError := base64.StdEncoding.DecodeString("123")
	tests := []struct {
		desc             string
		credUID          int
		secrets          map[string]string
		expectedFileName string
		expectedContent  []byte
		expectedErr      error
	}{
		{
			desc:    "[Success] Got correct filename and content",
			credUID: 1000,
			secrets: map[string]string{
				krb5CcacheName: base64Ticket,
			},
			expectedFileName: goodFileName,
			expectedContent:  ticket,
			expectedErr:      nil,
		},
		{
			desc:    "[Error] Throw error if credUID mismatch",
			credUID: 1001,
			secrets: map[string]string{
				krb5CcacheName: base64Ticket,
			},
			expectedFileName: "",
			expectedContent:  nil,
			expectedErr:      status.Error(codes.InvalidArgument, fmt.Sprintf("Empty kerberos cache in key %s", "krb5cc_1001")),
		},
		{
			desc:    "[Error] Throw error if ticket is empty in secret",
			credUID: 1000,
			secrets: map[string]string{
				krb5CcacheName: "",
			},
			expectedFileName: "",
			expectedContent:  nil,
			expectedErr:      status.Error(codes.InvalidArgument, fmt.Sprintf("Empty kerberos cache in key %s", krb5CcacheName)),
		},
		{
			desc:    "[Error] Throw error if ticket is invalid base64",
			credUID: 1000,
			secrets: map[string]string{
				krb5CcacheName: "123",
			},
			expectedFileName: "",
			expectedContent:  nil,
			expectedErr:      status.Error(codes.InvalidArgument, fmt.Sprintf("Malformed kerberos cache in key %s, expected to be in base64 form: %v", krb5CcacheName, base64DecError)),
		},
	}

	for _, test := range tests {
		fileName, content, err := getKerberosCache(krb5CacheDirectory, krb5Prefix, test.credUID, test.secrets)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("[%s]: Expected error : %v, Actual error: %v", test.desc, test.expectedErr, err)
		} else {
			if fileName != test.expectedFileName {
				t.Errorf("[%s]: Expected filename : %s, Actual result: %s", test.desc, test.expectedFileName, fileName)
			}
			if !reflect.DeepEqual(content, test.expectedContent) {
				t.Errorf("[%s]: Expected content : %s, Actual content: %s", test.desc, test.expectedContent, content)
			}
		}
	}

}

func TestNodePublishVolumeIdempotentMount(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() != 0 {
		return
	}
	sourceTest := "./sourcetest"
	err := makeDir(sourceTest)
	assert.NoError(t, err)

	targetTest := "./targettest"
	err = makeDir(targetTest)
	assert.NoError(t, err)

	d := NewFakeDriver()
	d.mounter = &mount.SafeFormatAndMount{
		Interface: mount.New(""),
		Exec:      exec.New(),
	}

	volumeCap := csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER}
	req := csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
		VolumeId:          "vol_1",
		TargetPath:        targetTest,
		StagingTargetPath: sourceTest,
		Readonly:          true}

	_, err = d.NodePublishVolume(context.Background(), &req)
	assert.NoError(t, err)
	_, err = d.NodePublishVolume(context.Background(), &req)
	assert.NoError(t, err)

	// ensure the target not be mounted twice
	targetAbs, err := filepath.Abs(targetTest)
	assert.NoError(t, err)

	mountList, err := d.mounter.List()
	assert.NoError(t, err)
	mountPointNum := 0
	for _, mountPoint := range mountList {
		if mountPoint.Path == targetAbs {
			mountPointNum++
		}
	}
	assert.Equal(t, 1, mountPointNum)
	err = d.mounter.Unmount(targetTest)
	assert.NoError(t, err)
	_ = d.mounter.Unmount(targetTest)
	err = os.RemoveAll(sourceTest)
	assert.NoError(t, err)
	err = os.RemoveAll(targetTest)
	assert.NoError(t, err)
}

func TestEnableGroupRWX(t *testing.T) {
	tests := []struct {
		value         string
		expectedValue string
	}{
		{
			value:         "qwerty",
			expectedValue: "qwerty",
		},
		{
			value:         "0111",
			expectedValue: "0171",
		},
	}

	for _, test := range tests {
		mode := enableGroupRWX(test.value)
		assert.Equal(t, test.expectedValue, mode)
	}
}

func TestRaiseGroupRWXInMountFlags(t *testing.T) {
	tests := []struct {
		mountFlags         []string
		flag               string
		expectedResult     bool
		mountFlagsUpdated  bool
		expectedMountFlags []string
	}{
		{
			mountFlags:     []string{""},
			flag:           "flag",
			expectedResult: false,
		},
		{
			mountFlags:     []string{"irrelevant"},
			flag:           "flag",
			expectedResult: false,
		},
		{
			mountFlags:     []string{"key=val"},
			flag:           "flag",
			expectedResult: false,
		},
		{
			mountFlags:     []string{"flag=key=val"},
			flag:           "flag",
			expectedResult: false,
		},
		{
			// This is important: if we return false here, the caller will append another flag=...
			mountFlags:     []string{"flag=invalid"},
			flag:           "flag",
			expectedResult: true,
		},
		{
			// Main case: raising group bits in the value
			mountFlags:         []string{"flag=0111"},
			flag:               "flag",
			expectedResult:     true,
			mountFlagsUpdated:  true,
			expectedMountFlags: []string{"flag=0171"},
		},
	}

	for _, test := range tests {
		savedMountFlags := make([]string, len(test.mountFlags))
		copy(savedMountFlags, test.mountFlags)

		result := raiseGroupRWXInMountFlags(test.mountFlags, test.flag)
		if result != test.expectedResult {
			t.Errorf("raiseGroupRWXInMountFlags(%v, %s) returned %t (expected: %t)",
				test.mountFlags, test.flag, result, test.expectedResult)
		}

		if test.mountFlagsUpdated {
			if !reflect.DeepEqual(test.expectedMountFlags, test.mountFlags) {
				t.Errorf("raiseGroupRWXInMountFlags(%v, %s) did not update mountFlags (expected: %v)",
					savedMountFlags, test.flag, test.expectedMountFlags)
			}
		} else {
			if !reflect.DeepEqual(savedMountFlags, test.mountFlags) {
				t.Errorf("raiseGroupRWXInMountFlags(%v, %s) updated mountFlags: %v",
					savedMountFlags, test.flag, test.mountFlags)
			}
		}
	}
}
