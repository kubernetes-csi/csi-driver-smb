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
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		req         csi.CreateVolumeRequest
		expectedErr error
	}{
		{
			desc:        "Volume capabilities missing",
			req:         csi.CreateVolumeRequest{},
			expectedErr: status.Error(codes.InvalidArgument, "CreateVolume Volume capabilities must be provided"),
		},
		{
			desc: "Valid request",
			req: csi.CreateVolumeRequest{
				VolumeCapabilities: stdVolCap,
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		_, err := d.CreateVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	d := NewFakeDriver()

	tests := []struct {
		desc        string
		req         csi.DeleteVolumeRequest
		expectedErr error
	}{
		{
			desc:        "Volume ID missing",
			req:         csi.DeleteVolumeRequest{},
			expectedErr: status.Error(codes.InvalidArgument, "Volume ID missing in request"),
		},
		{
			desc:        "Valid request",
			req:         csi.DeleteVolumeRequest{VolumeId: "vol_1"},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		_, err := d.DeleteVolume(context.Background(), &test.req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
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
