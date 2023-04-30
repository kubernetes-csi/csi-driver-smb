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
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fakeDriverName = "fake"
	fakeNodeID     = "fakeNodeID"
)

var (
	vendorVersion = "0.3.0"
)

func NewFakeDriver() *CSIDriver {

	driver := NewCSIDriver(fakeDriverName, vendorVersion, fakeNodeID)

	return driver
}

func TestNewFakeDriver(t *testing.T) {
	// Test New fake driver with invalid arguments.
	d := NewCSIDriver("", vendorVersion, fakeNodeID)
	assert.Nil(t, d)
}

func TestNewCSIDriver(t *testing.T) {
	tests := []struct {
		desc         string
		name         string
		version      string
		nodeID       string
		expectedResp *CSIDriver
	}{
		{
			desc:    "Successful",
			name:    fakeDriverName,
			version: vendorVersion,
			nodeID:  fakeNodeID,
			expectedResp: &CSIDriver{
				Name:    fakeDriverName,
				Version: vendorVersion,
				NodeID:  fakeNodeID,
			},
		},
		{
			desc:         "Missing driver name",
			name:         "",
			version:      vendorVersion,
			nodeID:       fakeNodeID,
			expectedResp: nil,
		},
		{
			desc:         "Missing node ID",
			name:         fakeDriverName,
			version:      vendorVersion,
			nodeID:       "",
			expectedResp: nil,
		},
		{
			desc:    "Missing driver version",
			name:    fakeDriverName,
			version: "",
			nodeID:  fakeNodeID,
			expectedResp: &CSIDriver{
				Name:    fakeDriverName,
				Version: "",
				NodeID:  fakeNodeID,
			},
		},
	}
	for _, test := range tests {
		resp := NewCSIDriver(test.name, test.version, test.nodeID)
		if !reflect.DeepEqual(resp, test.expectedResp) {
			t.Errorf("Unexpected driver: %v", resp)
		}
	}
}

func TestGetVolumeCapabilityAccessModes(t *testing.T) {

	d := NewFakeDriver()

	// Test no volume access modes.
	// REVISIT: Do we need to support any default access modes.
	c := d.GetVolumeCapabilityAccessModes()
	assert.Zero(t, len(c))

	// Test driver with access modes.
	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})
	modes := d.GetVolumeCapabilityAccessModes()
	assert.Equal(t, 1, len(modes))
	assert.Equal(t, modes[0].GetMode(), csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER)
}

func TestValidateControllerServiceRequest(t *testing.T) {
	d := NewFakeDriver()

	// Valid requests which require no capabilities
	err := d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_UNKNOWN)
	assert.NoError(t, err)

	// Test controller service publish/unpublish not supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME)
	s, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, s.Code(), codes.InvalidArgument)

	// Add controller service publish & unpublish request
	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_GET_CAPACITY,
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
			csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		})

	// Test controller service publish/unpublish is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME)
	assert.NoError(t, err)

	// Test controller service create/delete is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME)
	assert.NoError(t, err)

	// Test controller service list volumes is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_LIST_VOLUMES)
	assert.NoError(t, err)

	// Test controller service get capacity is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_GET_CAPACITY)
	assert.NoError(t, err)

	// Test controller service clone volumes is supported
	err = d.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CLONE_VOLUME)
	assert.NoError(t, err)

}

func TestValidateNodeServiceRequest(t *testing.T) {
	d := NewFakeDriver()
	d.NSCap = []*csi.NodeServiceCapability{
		NewNodeServiceCapability(csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME),
		NewNodeServiceCapability(csi.NodeServiceCapability_RPC_GET_VOLUME_STATS),
	}
	tests := []struct {
		desc        string
		cap         csi.NodeServiceCapability_RPC_Type
		expectedErr error
	}{
		{
			desc:        "Node service capabailtiy unknown",
			cap:         csi.NodeServiceCapability_RPC_UNKNOWN,
			expectedErr: nil,
		},
		{
			desc:        "Successful request",
			cap:         csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
			expectedErr: nil,
		},
		{
			desc:        "Invalid argument",
			cap:         csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
			expectedErr: status.Error(codes.InvalidArgument, "EXPAND_VOLUME"),
		},
	}

	for _, test := range tests {
		err := d.ValidateNodeServiceRequest(test.cap)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

func TestAddControllerServiceCapabilities(t *testing.T) {
	d := NewFakeDriver()
	expectedCapList := []*csi.ControllerServiceCapability{
		NewControllerServiceCapability(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME),
		NewControllerServiceCapability(csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME),
	}
	capList := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
	}
	d.AddControllerServiceCapabilities(capList)
	assert.Equal(t, d.Cap, expectedCapList)
}

func TestAddNodeServiceCapabilities(t *testing.T) {
	d := NewFakeDriver()
	expectedCapList := []*csi.NodeServiceCapability{
		NewNodeServiceCapability(csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME),
		NewNodeServiceCapability(csi.NodeServiceCapability_RPC_GET_VOLUME_STATS),
	}
	capList := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
	}
	d.AddNodeServiceCapabilities(capList)
	assert.Equal(t, d.NSCap, expectedCapList)
}

func TestAddVolumeCapabilityAccessModes(t *testing.T) {
	d := NewFakeDriver()
	expectedCapList := []*csi.VolumeCapability_AccessMode{
		NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER),
		NewVolumeCapabilityAccessMode(csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY),
	}
	capList := []csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
	}
	d.AddVolumeCapabilityAccessModes(capList)
	assert.Equal(t, d.VC, expectedCapList)
}
