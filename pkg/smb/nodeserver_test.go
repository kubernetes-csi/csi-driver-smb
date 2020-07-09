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
	"syscall"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/mount"
)

const (
	sourceTest = "./source_test"
	targetTest = "./target_test"
)

func TestNodeStageVolume(t *testing.T) {
	stdVolCap := csi.VolumeCapability{
		AccessType: &csi.VolumeCapability_Mount{
			Mount: &csi.VolumeCapability_MountVolume{},
		},
	}

	errorMountSensSource := "./error_mount_sens_source"
	smbFile := "./smb.go"

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
		req         csi.NodeStageVolumeRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume ID missing",
			req:         csi.NodeStageVolumeRequest{},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc:        "[Error] Volume capabilities missing",
			req:         csi.NodeStageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "Volume capability not provided"),
		},
		{
			desc:        "[Error] Stage target path missing",
			req:         csi.NodeStageVolumeRequest{VolumeId: "vol_1", VolumeCapability: &stdVolCap},
			expectedErr: status.Error(codes.InvalidArgument, "Staging target not provided"),
		},
		{
			desc: "[Error] Source field is missing in context",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap},
			expectedErr: status.Error(codes.InvalidArgument, "source field is missing, current context: map[]"),
		},
		{
			desc: "[Error] Not a Directory",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: smbFile,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: status.Error(codes.Internal, "MkdirAll ./smb.go failed with error: mkdir ./smb.go: not a directory"),
		},
		{
			desc: "[Error] Failed SMB mount mocked by MountSensitive",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: errorMountSensSource,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: status.Errorf(codes.Internal, "volume(vol_1##) mount \"test_source\" on \"./error_mount_sens_source\" failed with fake MountSensitive: target error"),
		},
		{
			desc: "[Success] Valid request",
			req: csi.NodeStageVolumeRequest{VolumeId: "vol_1##", StagingTargetPath: sourceTest,
				VolumeCapability: &stdVolCap,
				VolumeContext:    volContext,
				Secrets:          secrets},
			expectedErr: nil,
		},
	}

	// Setup
	d := NewFakeDriver()

	for _, test := range tests {
		fakeMounter := &fakeMounter{}
		d.mounter = &mount.SafeFormatAndMount{
			Interface: fakeMounter,
		}

		_, err := d.NodeStageVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("test case: %s, Unexpected error: %v", test.desc, err)
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

func TestNodeGetVolumeStats(t *testing.T) {
	d := NewFakeDriver()
	req := csi.NodeGetVolumeStatsRequest{}
	resp, err := d.NodeGetVolumeStats(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
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
	errorMountSource := "./error_mount_source"
	alreadyMountedTarget := "./false_is_likely_exist_target"
	smbFile := "./smb.go"

	tests := []struct {
		desc        string
		req         csi.NodePublishVolumeRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume capabilities missing",
			req:         csi.NodePublishVolumeRequest{},
			expectedErr: status.Error(codes.InvalidArgument, "Volume capability missing in request"),
		},
		{
			desc:        "[Error] Volume ID missing",
			req:         csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap}},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc: "[Error] Target path missing",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "Target path not provided"),
		},
		{
			desc: "[Error] Stage target path missing",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:   "vol_1",
				TargetPath: targetTest},
			expectedErr: status.Error(codes.InvalidArgument, "Staging target not provided"),
		},
		{
			desc: "[Error] Not a directory",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        smbFile,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: status.Errorf(codes.Internal, "Could not mount target \"./smb.go\": mkdir ./smb.go: not a directory"),
		},
		{
			desc: "[Error] Mount error mocked by Mount",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: errorMountSource,
				Readonly:          true},
			expectedErr: status.Errorf(codes.Internal, "Could not mount \"./error_mount_source\" at \"./target_test\": fake Mount: source error"),
		},
		{
			desc: "[Success] Valid request read only",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: nil,
		},
		{
			desc: "[Success] Valid request already mounted",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        alreadyMountedTarget,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: nil,
		},
		{
			desc: "[Success] Valid request",
			req: csi.NodePublishVolumeRequest{VolumeCapability: &csi.VolumeCapability{AccessMode: &volumeCap},
				VolumeId:          "vol_1",
				TargetPath:        targetTest,
				StagingTargetPath: sourceTest,
				Readonly:          true},
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(alreadyMountedTarget)
	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}

	for _, test := range tests {
		_, err := d.NodePublishVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("test case: %s, Unexpected error: %v", test.desc, err)
		}
	}

	// Clean up
	err := os.RemoveAll(targetTest)
	assert.NoError(t, err)
	err = os.RemoveAll(alreadyMountedTarget)
	assert.NoError(t, err)
}

func TestNodeUnpublishVolume(t *testing.T) {
	errorTarget := "./error_is_likely_target"
	targetFile := "./abc.go"

	tests := []struct {
		desc        string
		req         csi.NodeUnpublishVolumeRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume ID missing",
			req:         csi.NodeUnpublishVolumeRequest{TargetPath: targetTest},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc:        "[Error] Target missing",
			req:         csi.NodeUnpublishVolumeRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "Target path missing in request"),
		},
		{
			desc:        "[Error] Unmount error mocked by IsLikelyNotMountPoint",
			req:         csi.NodeUnpublishVolumeRequest{TargetPath: errorTarget, VolumeId: "vol_1"},
			expectedErr: status.Error(codes.Internal, "failed to unmount target \"./error_is_likely_target\": fake IsLikelyNotMountPoint: fake error"),
		},
		{
			desc:        "[Success] Valid request",
			req:         csi.NodeUnpublishVolumeRequest{TargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}

	for _, test := range tests {
		_, err := d.NodeUnpublishVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("test case: %s, Unexpected error: %v", test.desc, err)
		}
	}

	// Clean up
	err := os.RemoveAll(errorTarget)
	assert.NoError(t, err)
}

func TestNodeUnstageVolume(t *testing.T) {
	errorTarget := "./error_is_likely_target"
	targetFile := "./abc.go"

	tests := []struct {
		desc        string
		req         csi.NodeUnstageVolumeRequest
		expectedErr error
	}{
		{
			desc:        "[Error] Volume ID missing",
			req:         csi.NodeUnstageVolumeRequest{StagingTargetPath: targetTest},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc:        "[Error] Target missing",
			req:         csi.NodeUnstageVolumeRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "Staging target not provided"),
		},
		{
			desc:        "[Error] CleanupMountPoint error mocked by IsLikelyNotMountPoint",
			req:         csi.NodeUnstageVolumeRequest{StagingTargetPath: errorTarget, VolumeId: "vol_1"},
			expectedErr: status.Error(codes.Internal, "failed to unmount staging target \"./error_is_likely_target\": fake IsLikelyNotMountPoint: fake error"),
		},
		{
			desc:        "[Success] Valid request",
			req:         csi.NodeUnstageVolumeRequest{StagingTargetPath: targetFile, VolumeId: "vol_1"},
			expectedErr: nil,
		},
	}

	// Setup
	_ = makeDir(errorTarget)
	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}

	for _, test := range tests {
		_, err := d.NodeUnstageVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("test case: %s, Unexpected error: %v", test.desc, err)
		}
	}

	// Clean up
	err := os.RemoveAll(errorTarget)
	assert.NoError(t, err)
}

func TestEnsureMountPoint(t *testing.T) {
	errorTarget := "./error_is_likely_target"
	alreadyExistTarget := "./false_is_likely_exist_target"
	falseTarget := "./false_is_likely_target"
	smbFile := "./smb.go"

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
