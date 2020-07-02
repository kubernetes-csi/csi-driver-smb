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

package csicommon

import (
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidateVolumeCapabilities(t *testing.T) {
	d := NewFakeDriver()
	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
	})

	capability := []*csi.VolumeCapability{
		{AccessMode: NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER)},
		{AccessMode: NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY)},
		{AccessMode: NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY)},
	}
	capabilityDisjoint := []*csi.VolumeCapability{
		{AccessMode: NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER)},
	}

	ns := NewDefaultControllerServer(d)

	// Test when there are common capabilities
	req := csi.ValidateVolumeCapabilitiesRequest{VolumeCapabilities: capability}
	resp, err := ns.ValidateVolumeCapabilities(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, resp.XXX_sizecache, int32(0))

	// Test when there are no common capabilities
	req = csi.ValidateVolumeCapabilitiesRequest{VolumeCapabilities: capabilityDisjoint}
	resp, err = ns.ValidateVolumeCapabilities(context.Background(), &req)
	assert.NotNil(t, resp)
	assert.Error(t, err)
}

func TestControllerGetCapabilities(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)

	// Test valid request
	req := csi.ControllerGetCapabilitiesRequest{}
	resp, err := ns.ControllerGetCapabilities(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, resp.XXX_sizecache, int32(0))
}

func TestCreateVolume(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.CreateVolumeRequest{}
	resp, err := ns.CreateVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteVolume(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.DeleteVolumeRequest{}
	resp, err := ns.DeleteVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerPublishVolume(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.ControllerPublishVolumeRequest{}
	resp, err := ns.ControllerPublishVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.ControllerUnpublishVolumeRequest{}
	resp, err := ns.ControllerUnpublishVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetCapacity(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.GetCapacityRequest{}
	resp, err := ns.GetCapacity(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListVolumes(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.ListVolumesRequest{}
	resp, err := ns.ListVolumes(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCreateSnapshot(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.CreateSnapshotRequest{}
	resp, err := ns.CreateSnapshot(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.DeleteSnapshotRequest{}
	resp, err := ns.DeleteSnapshot(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListSnapshots(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.ListSnapshotsRequest{}
	resp, err := ns.ListSnapshots(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestControllerExpandVolume(t *testing.T) {
	d := NewFakeDriver()
	ns := NewDefaultControllerServer(d)
	req := csi.ControllerExpandVolumeRequest{}
	resp, err := ns.ControllerExpandVolume(context.Background(), &req)
	assert.Nil(t, resp)
	if !reflect.DeepEqual(err, status.Error(codes.Unimplemented, "")) {
		t.Errorf("Unexpected error: %v", err)
	}
}
