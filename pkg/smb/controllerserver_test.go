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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-driver-smb/test/utils/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testServer    = "test-server/baseDir"
	testCSIVolume = "test-csi"
	testVolumeID  = "test-server/baseDir#test-csi"
)

func TestControllerGetCapabilities(t *testing.T) {
	d := NewFakeDriver()
	controlCap := []*csi.ControllerServiceCapability{
		{
			Type: &csi.ControllerServiceCapability_Rpc{},
		},
	}
	d.Cap = controlCap
	req := csi.ControllerGetCapabilitiesRequest{}
	resp, err := d.ControllerGetCapabilities(context.Background(), &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.Capabilities, controlCap)
}

func TestCreateVolume(t *testing.T) {
	d := NewFakeDriver()

	// Setup workingMountDir
	workingMountDir, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory")
	}
	d.workingMountDir = workingMountDir

	// Setup mounter
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
	}
	d.mounter = mounter

	sourceTest := testutil.GetWorkDirPath("test-csi", t)

	cases := []struct {
		name                     string
		req                      *csi.CreateVolumeRequest
		resp                     *csi.CreateVolumeResponse
		flakyWindowsErrorMessage string
		expectErr                bool
	}{
		{
			name: "valid defaults",
			req: &csi.CreateVolumeRequest{
				Name: testCSIVolume,
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					paramSource: testServer,
				},
				Secrets: map[string]string{
					usernameField: "test",
					passwordField: "test",
					domainField:   "test_doamin",
				},
			},
			resp: &csi.CreateVolumeResponse{
				Volume: &csi.Volume{
					VolumeId: testVolumeID,
					VolumeContext: map[string]string{
						paramSource: filepath.Join(testServer, testCSIVolume),
					},
				},
			},
			flakyWindowsErrorMessage: fmt.Sprintf("volume(vol_1##) mount \"test-server\" on %#v failed with "+
				"smb mapping failed with error: rpc error: code = Unknown desc = NewSmbGlobalMapping failed.",
				sourceTest),
		},
		{
			name: "name empty",
			req: &csi.CreateVolumeRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					paramSource: testServer,
				},
			},
			expectErr: true,
		},
		{
			name: "invalid create context",
			req: &csi.CreateVolumeRequest{
				Name: testCSIVolume,
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
				Parameters: map[string]string{
					"unknown-parameter": "foo",
				},
			},
			expectErr: true,
		},
	}

	for _, test := range cases {
		test := test //pin
		t.Run(test.name, func(t *testing.T) {
			// Setup
			_ = os.MkdirAll(filepath.Join(d.workingMountDir, testCSIVolume), os.ModePerm)

			// Run
			resp, err := d.CreateVolume(context.TODO(), test.req)

			// Verify
			if test.expectErr && err == nil {
				t.Errorf("test %q failed; got success", test.name)
			}

			// separate assertion for flaky error messages
			if test.flakyWindowsErrorMessage != "" && runtime.GOOS == "windows" {
				fmt.Println("Skipping checks on Windows ENV")
			} else {
				if !test.expectErr && err != nil {
					t.Errorf("test %q failed: %v", test.name, err)
				}
				if !reflect.DeepEqual(resp, test.resp) {
					t.Errorf("test %q failed: got resp %+v, expected %+v", test.name, resp, test.resp)
				}
				if !test.expectErr {
					info, err := os.Stat(filepath.Join(d.workingMountDir, test.req.Name, test.req.Name))
					if err != nil {
						t.Errorf("test %q failed: couldn't find volume subdirectory: %v", test.name, err)
					}
					if !info.IsDir() {
						t.Errorf("test %q failed: subfile not a directory", test.name)
					}
				}
			}
		})
	}
}

func TestDeleteVolume(t *testing.T) {
	d := NewFakeDriver()

	// Setup workingMountDir
	workingMountDir, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get current working directory")
	}
	d.workingMountDir = workingMountDir

	// Setup mounter
	mounter, err := NewFakeMounter()
	if err != nil {
		t.Fatalf(fmt.Sprintf("failed to get fake mounter: %v", err))
	}
	d.mounter = mounter

	cases := []struct {
		desc        string
		req         *csi.DeleteVolumeRequest
		resp        *csi.DeleteVolumeResponse
		expectedErr error
	}{
		{
			desc:        "Volume ID missing",
			req:         &csi.DeleteVolumeRequest{},
			resp:        nil,
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc: "Valid request",
			req: &csi.DeleteVolumeRequest{
				VolumeId: testVolumeID,
				Secrets: map[string]string{
					usernameField: "test",
					passwordField: "test",
					domainField:   "test_doamin",
				},
			},
			resp:        &csi.DeleteVolumeResponse{},
			expectedErr: nil,
		},
	}
	for _, test := range cases {
		test := test //pin
		t.Run(test.desc, func(t *testing.T) {
			// Setup
			_ = os.MkdirAll(filepath.Join(d.workingMountDir, testCSIVolume), os.ModePerm)
			_, _ = os.Create(filepath.Join(d.workingMountDir, testCSIVolume, testCSIVolume))
			// Run
			resp, err := d.DeleteVolume(context.TODO(), test.req)
			// Verify
			if runtime.GOOS == "windows" {
				// skip checks
				fmt.Println("Skipping checks on Windows ENV")
			} else {
				if test.expectedErr == nil && err != nil {
					t.Errorf("test %q failed: %v", test.desc, err)
				}
				if test.expectedErr != nil && err == nil {
					t.Errorf("test %q failed; expected error %v, got success", test.desc, test.expectedErr)
				}
				if !reflect.DeepEqual(resp, test.resp) {
					t.Errorf("test %q failed: got resp %+v, expected %+v", test.desc, resp, test.resp)
				}
				if _, err := os.Stat(filepath.Join(d.workingMountDir, testCSIVolume, testCSIVolume)); test.expectedErr == nil && !os.IsNotExist(err) {
					t.Errorf("test %q failed: expected volume subdirectory deleted, it still exists", test.desc)
				}
			}
		})
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {
	d := NewFakeDriver()
	stdVolCap := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	tests := []struct {
		desc        string
		req         csi.ValidateVolumeCapabilitiesRequest
		expectedErr error
	}{
		{
			desc:        "Volume ID missing",
			req:         csi.ValidateVolumeCapabilitiesRequest{},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc:        "Volume capabilities missing",
			req:         csi.ValidateVolumeCapabilitiesRequest{VolumeId: "vol_1"},
			expectedErr: status.Error(codes.InvalidArgument, "Volume capabilities missing in request"),
		},
		{
			desc: "Valid request",
			req: csi.ValidateVolumeCapabilitiesRequest{
				VolumeId:           "vol_1#f5713de20cde511e8ba4900#fileshare#diskname#",
				VolumeCapabilities: stdVolCap,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		_, err := d.ValidateVolumeCapabilities(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestControllerPublishVolume(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ControllerPublishVolumeRequest{}
	resp, err := d.ControllerPublishVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ControllerUnpublishVolumeRequest{}
	resp, err := d.ControllerUnpublishVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetCapacity(t *testing.T) {
	d := NewFakeDriver()
	req := csi.GetCapacityRequest{}
	resp, err := d.GetCapacity(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListVolumes(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ListVolumesRequest{}
	resp, err := d.ListVolumes(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerExpandVolume(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ControllerExpandVolumeRequest{}
	resp, err := d.ControllerExpandVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerGetVolume(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ControllerGetVolumeRequest{}
	resp, err := d.ControllerGetVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCreateSnapshot(t *testing.T) {
	d := NewFakeDriver()
	req := csi.CreateSnapshotRequest{}
	resp, err := d.CreateSnapshot(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	d := NewFakeDriver()
	req := csi.DeleteSnapshotRequest{}
	resp, err := d.DeleteSnapshot(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListSnapshots(t *testing.T) {
	d := NewFakeDriver()
	req := csi.ListSnapshotsRequest{}
	resp, err := d.ListSnapshots(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetSmbVolFromID(t *testing.T) {

	cases := []struct {
		desc      string
		volumeID  string
		source    string
		subDir    string
		expectErr bool
	}{
		{
			desc:      "correct volume id",
			volumeID:  "smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			expectErr: false,
		},
		{
			desc:      "correct volume id with multiple base directories",
			volumeID:  "smb-server.default.svc.cluster.local/share/dir1/dir2#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			source:    "//smb-server.default.svc.cluster.local/share/dir1/dir2",
			subDir:    "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			expectErr: false,
		},
		{
			desc:      "incorrect volume id",
			volumeID:  "smb-server.default.svc.cluster.local/share",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			expectErr: true,
		},
	}
	for _, test := range cases {
		test := test //pin
		t.Run(test.desc, func(t *testing.T) {
			smbVolume, err := getSmbVolFromID(test.volumeID)

			if !test.expectErr {
				assert.Equal(t, smbVolume.sourceField, test.source)
				assert.Equal(t, smbVolume.subDir, test.subDir)
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
