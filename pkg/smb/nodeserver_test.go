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
	"bytes"
	"context"
	"flag"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog"
	"testing"
)

func TestNodeStageVolume(t *testing.T) {

	klog.InitFlags(nil)
	if e := flag.Set("logtostderr", "false"); e != nil {
		t.Error(e)
	}
	if e := flag.Set("alsologtostderr", "false"); e != nil {
		t.Error(e)
	}
	if e := flag.Set("v", "100"); e != nil {
		t.Error(e)
	}
	flag.Parse()

	buf := new(bytes.Buffer)
	klog.SetOutput(buf)

	d := NewFakeDriver()

	tests := []struct {
		name   string
		req    *csi.NodeStageVolumeRequest
		expStr string
	}{
		{
			"with secrets",
			&csi.NodeStageVolumeRequest{
				VolumeId: "vol_1",
				Secrets: map[string]string{
					"password": "testpassword",
					"username": "testuser",
				},
				VolumeCapability: &csi.VolumeCapability{},
				XXX_sizecache:    100,
			},
			`NodeStageVolume called with request {vol_1 map[]   map[password:**** username:testuser] map[] {} [] 100}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// EXECUTE
			_, _ = d.NodeStageVolume(context.Background(), test.req)
			klog.Flush()

			//ASSERT
			assert.Contains(t, buf.String(), test.expStr)

			// CLEANUP
			buf.Reset()
		})
	}

}
