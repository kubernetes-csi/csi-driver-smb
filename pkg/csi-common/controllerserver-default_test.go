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
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
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
