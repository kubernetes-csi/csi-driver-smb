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
	testVolumeID  = "test-server/baseDir#test-csi##"
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

	blockVolCap := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Block{
				Block: &csi.VolumeCapability_BlockVolume{},
			},
		},
	}

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
		skipOnWindows            bool
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
					sourceField: testServer,
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
						sourceField: testServer,
						subDirField: testCSIVolume,
					},
				},
			},
			skipOnWindows: true,
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
					sourceField: testServer,
				},
			},
			expectErr: true,
		},
		{
			name: "Volume capabilities missing",
			req: &csi.CreateVolumeRequest{
				VolumeCapabilities: []*csi.VolumeCapability{},
			},
			expectErr: true,
		},
		{
			name: "block volume capability not supported",
			req: &csi.CreateVolumeRequest{
				VolumeCapabilities: blockVolCap,
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
			if test.skipOnWindows && runtime.GOOS == "windows" {
				return
			}
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
				fmt.Println("Skipping checks on Windows ENV") // nolint
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
				fmt.Println("Skipping checks on Windows ENV") // nolint
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
	mountVolCap := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	blockVolCap := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Block{
				Block: &csi.VolumeCapability_BlockVolume{},
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
			expectedErr: status.Error(codes.InvalidArgument, "volume capabilities missing in request"),
		},
		{
			desc: "block volume capability not supported",
			req: csi.ValidateVolumeCapabilitiesRequest{
				VolumeId:           "vol_1",
				VolumeCapabilities: blockVolCap,
			},
			expectedErr: status.Error(codes.InvalidArgument, "block volume capability not supported"),
		},
		{
			desc: "Valid request",
			req: csi.ValidateVolumeCapabilitiesRequest{
				VolumeId:           "vol_1#f5713de20cde511e8ba4900#fileshare#diskname#",
				VolumeCapabilities: mountVolCap,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		_, err := d.ValidateVolumeCapabilities(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("[test: %s] Unexpected error: %v, expected error: %v", test.desc, err, test.expectedErr)
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
		uuid      string
		onDelete  string
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
			desc:      "correct volume id with //",
			volumeID:  "//smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			expectErr: false,
		},
		{
			desc:      "correct volume id with empty uuid",
			volumeID:  "smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f#",
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
			desc:      "existing sub dir",
			volumeID:  "smb-server.default.svc.cluster.local/share#subdir#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "subdir",
			uuid:      "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			expectErr: false,
		},
		{
			desc:      "valid request nested ondelete retain",
			volumeID:  "smb-server.default.svc.cluster.local/share#subdir#pvc-4729891a-f57e-4982-9c60-e9884af1be2f#retain",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "subdir",
			uuid:      "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			onDelete:  "retain",
			expectErr: false,
		},
		{
			desc:      "valid request nested ondelete archive",
			volumeID:  "smb-server.default.svc.cluster.local/share#subdir#pvc-4729891a-f57e-4982-9c60-e9884af1be2f#archive",
			source:    "//smb-server.default.svc.cluster.local/share",
			subDir:    "subdir",
			uuid:      "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			onDelete:  "archive",
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
				assert.Equal(t, smbVolume.source, test.source)
				assert.Equal(t, smbVolume.subDir, test.subDir)
				assert.Equal(t, smbVolume.uuid, test.uuid)
				assert.Equal(t, smbVolume.onDelete, test.onDelete)
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestGetVolumeIDFromSmbVol(t *testing.T) {
	cases := []struct {
		desc   string
		vol    *smbVolume
		result string
	}{
		{
			desc: "volume without uuid",
			vol: &smbVolume{
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "subdir",
			},
			result: "smb-server.default.svc.cluster.local/share#subdir##",
		},
		{
			desc: "volume with uuid",
			vol: &smbVolume{
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "subdir",
				uuid:   "uuid",
			},
			result: "smb-server.default.svc.cluster.local/share#subdir#uuid#",
		},
		{
			desc: "volume without subdir",
			vol: &smbVolume{
				source: "//smb-server.default.svc.cluster.local/share",
			},
			result: "smb-server.default.svc.cluster.local/share###",
		},
		{
			desc: "volume with nested onDelete retain",
			vol: &smbVolume{
				source:   "//smb-server.default.svc.cluster.local/share",
				subDir:   "subdir",
				uuid:     "uuid",
				onDelete: "retain",
			},
			result: "smb-server.default.svc.cluster.local/share#subdir#uuid#retain",
		},
	}

	for _, test := range cases {
		volumeID := getVolumeIDFromSmbVol(test.vol)
		assert.Equal(t, volumeID, test.result)
	}
}

func TestGetInternalMountPath(t *testing.T) {
	cases := []struct {
		desc            string
		workingMountDir string
		vol             *smbVolume
		result          string
	}{
		{
			desc:            "nil volume",
			workingMountDir: "/tmp",
			result:          "",
		},
		{
			desc:            "uuid not empty",
			workingMountDir: "/tmp",
			vol: &smbVolume{
				subDir: "subdir",
				uuid:   "uuid",
			},
			result: filepath.Join("/tmp", "uuid"),
		},
		{
			desc:            "uuid empty",
			workingMountDir: "/tmp",
			vol: &smbVolume{
				subDir: "subdir",
				uuid:   "",
			},
			result: filepath.Join("/tmp", "subdir"),
		},
	}

	for _, test := range cases {
		path := getInternalMountPath(test.workingMountDir, test.vol)
		assert.Equal(t, path, test.result)
	}
}

func TestNewSMBVolume(t *testing.T) {
	cases := []struct {
		desc      string
		name      string
		size      int64
		params    map[string]string
		expectVol *smbVolume
		expectErr error
	}{
		{
			desc: "subDir is specified",
			name: "pv-name",
			size: 100,
			params: map[string]string{
				"source": "//smb-server.default.svc.cluster.local/share",
				"subDir": "subdir",
			},
			expectVol: &smbVolume{
				id:     "smb-server.default.svc.cluster.local/share#subdir#pv-name#",
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "subdir",
				size:   100,
				uuid:   "pv-name",
			},
		},
		{
			desc: "subDir with pv/pvc metadata is specified",
			name: "pv-name",
			size: 100,
			params: map[string]string{
				"source":        "//smb-server.default.svc.cluster.local/share",
				"subDir":        fmt.Sprintf("subdir-%s-%s-%s", pvcNameMetadata, pvcNamespaceMetadata, pvNameMetadata),
				pvcNameKey:      "pvcname",
				pvcNamespaceKey: "pvcnamespace",
				pvNameKey:       "pvname",
			},
			expectVol: &smbVolume{
				id:     "smb-server.default.svc.cluster.local/share#subdir-pvcname-pvcnamespace-pvname#pv-name#",
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "subdir-pvcname-pvcnamespace-pvname",
				size:   100,
				uuid:   "pv-name",
			},
		},
		{
			desc: "subDir not specified",
			name: "pv-name",
			size: 200,
			params: map[string]string{
				"source": "//smb-server.default.svc.cluster.local/share",
			},
			expectVol: &smbVolume{
				id:     "smb-server.default.svc.cluster.local/share#pv-name##",
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "pv-name",
				size:   200,
				uuid:   "",
			},
		},
		{
			desc:      "invalid parameter",
			params:    map[string]string{"invalid-parameter": "value"},
			expectVol: nil,
			expectErr: fmt.Errorf("invalid parameter %s in storage class", "invalid-parameter"),
		},
		{
			desc:      "source value is empty",
			params:    map[string]string{},
			expectVol: nil,
			expectErr: fmt.Errorf("%s is a required parameter", sourceField),
		},
	}

	for _, test := range cases {
		vol, err := newSMBVolume(test.name, test.size, test.params, "")
		if !reflect.DeepEqual(err, test.expectErr) {
			t.Errorf("[test: %s] Unexpected error: %v, expected error: %v", test.desc, err, test.expectErr)
		}
		if !reflect.DeepEqual(vol, test.expectVol) {
			t.Errorf("[test: %s] Unexpected vol: %v, expected vol: %v", test.desc, vol, test.expectVol)
		}
	}
}

func TestIsValidVolumeCapabilities(t *testing.T) {
	mountVolumeCapabilities := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{},
			},
		},
	}
	blockVolumeCapabilities := []*csi.VolumeCapability{
		{
			AccessType: &csi.VolumeCapability_Block{
				Block: &csi.VolumeCapability_BlockVolume{},
			},
		},
	}

	cases := []struct {
		desc      string
		volCaps   []*csi.VolumeCapability
		expectErr error
	}{
		{
			volCaps:   mountVolumeCapabilities,
			expectErr: nil,
		},
		{
			volCaps:   blockVolumeCapabilities,
			expectErr: fmt.Errorf("block volume capability not supported"),
		},
		{
			volCaps:   []*csi.VolumeCapability{},
			expectErr: fmt.Errorf("volume capabilities missing in request"),
		},
	}

	for _, test := range cases {
		err := isValidVolumeCapabilities(test.volCaps)
		if !reflect.DeepEqual(err, test.expectErr) {
			t.Errorf("[test: %s] Unexpected error: %v, expected error: %v", test.desc, err, test.expectErr)
		}
	}
}

func TestCopyFromVolume(t *testing.T) {
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
		desc      string
		req       *csi.CreateVolumeRequest
		dstVol    *smbVolume
		expectErr error
	}{
		{
			desc: "getInternalVolumePath failed",
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
					sourceField: testServer,
				},
				Secrets: map[string]string{
					usernameField: "test",
					passwordField: "test",
					domainField:   "test_doamin",
				},
				VolumeContentSource: &csi.VolumeContentSource{
					Type: &csi.VolumeContentSource_Volume{
						Volume: &csi.VolumeContentSource_VolumeSource{
							VolumeId: "unit-test",
						},
					},
				},
			},
			dstVol: &smbVolume{
				id:     "smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			},
			expectErr: status.Error(codes.NotFound, "could not split \"unit-test\" into server and subDir"),
		},
		{
			desc: "valid copy",
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
					sourceField: testServer,
				},
				Secrets: map[string]string{
					usernameField: "test",
					passwordField: "test",
					domainField:   "test_doamin",
				},
				VolumeContentSource: &csi.VolumeContentSource{
					Type: &csi.VolumeContentSource_Volume{
						Volume: &csi.VolumeContentSource_VolumeSource{
							VolumeId: testVolumeID,
						},
					},
				},
			},
			dstVol: &smbVolume{
				id:     "smb-server.default.svc.cluster.local/share#pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
				source: "//smb-server.default.svc.cluster.local/share",
				subDir: "pvc-4729891a-f57e-4982-9c60-e9884af1be2f",
			},
			expectErr: nil,
		},
	}

	for _, test := range cases {
		test := test //pin
		t.Run(test.desc, func(t *testing.T) {
			// Setup
			_ = os.MkdirAll(filepath.Join(d.workingMountDir, testCSIVolume, testCSIVolume), os.ModePerm)

			err := d.copyFromVolume(context.TODO(), test.req, test.dstVol)
			if runtime.GOOS == "windows" {
				fmt.Println("Skipping checks on Windows ENV") // nolint
			} else {
				if !reflect.DeepEqual(err, test.expectErr) {
					t.Errorf("[test: %s] Unexpected error: %v, expected error: %v", test.desc, err, test.expectErr)
				}
			}
		})
	}
}
