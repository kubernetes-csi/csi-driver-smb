/*
Copyright 2019 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-sigs/azurefile-csi-driver/pkg/azurefile"
	"github.com/kubernetes-sigs/azurefile-csi-driver/test/e2e/driver"
	"github.com/kubernetes-sigs/azurefile-csi-driver/test/e2e/testsuites"
	. "github.com/onsi/ginkgo"

	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	defaultDiskSize = 10

	dummyVolumeName = "pre-provisioned"
)

var (
	defaultDiskSizeBytes int64 = defaultDiskSize * 1024 * 1024 * 1024
)

var _ = Describe("[azurefile-csi-e2e] [single-az] Pre-Provisioned", func() {
	f := framework.NewDefaultFramework("azurefile")

	var (
		cs         clientset.Interface
		ns         *v1.Namespace
		testDriver driver.PreProvisionedVolumeTestDriver
		volumeID   string
		diskSize   string
		// Set to true if the volume should be deleted automatically after test
		skipManuallyDeletingVolume bool
	)

	nodeid := os.Getenv("nodeid")
	azurefileDriver := azurefile.NewDriver(nodeid)
	endpoint := "unix:///tmp/csi.sock"

	go func() {
		azurefileDriver.Run(endpoint)
	}()

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		testDriver = driver.InitAzureFileCSIDriver()

		req := &csi.CreateVolumeRequest{
			Name: dummyVolumeName,
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			CapacityRange: &csi.CapacityRange{
				RequiredBytes: defaultDiskSizeBytes,
				LimitBytes:    defaultDiskSizeBytes,
			},
		}
		resp, err := azurefileDriver.CreateVolume(context.Background(), req)
		if err != nil {
			Fail(fmt.Sprintf("create volume error: %v", err))
		}

		volumeID = resp.Volume.VolumeId
		diskSize = fmt.Sprintf("%dGi", defaultDiskSize)
		By(fmt.Sprintf("Successfully provisioned AzureFile volume: %q\n", volumeID))
	})

	AfterEach(func() {
		if !skipManuallyDeletingVolume {
			req := &csi.DeleteVolumeRequest{
				VolumeId: volumeID,
			}
			_, err := azurefileDriver.DeleteVolume(context.Background(), req)
			if err != nil {
				Fail(fmt.Sprintf("create volume %q error: %v", volumeID, err))
			}
		}
	})

	It("[env] should use a pre-provisioned volume and mount it as readOnly in a pod", func() {
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeID:  volumeID,
						FSType:    "ext4",
						ClaimSize: diskSize,
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
							ReadOnly:          true,
						},
					},
				},
			},
		}
		test := testsuites.PreProvisionedReadOnlyVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})
})
