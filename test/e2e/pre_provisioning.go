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

	"sigs.k8s.io/azurefile-csi-driver/test/e2e/driver"
	"sigs.k8s.io/azurefile-csi-driver/test/e2e/testsuites"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

const (
	defaultDiskSize      = 10
	defaultDiskSizeBytes = defaultDiskSize * 1024 * 1024 * 1024
)

var _ = ginkgo.Describe("[azurefile-csi-e2e] [single-az] Pre-Provisioned", func() {
	f := framework.NewDefaultFramework("azurefile")

	var (
		cs         clientset.Interface
		ns         *v1.Namespace
		testDriver driver.PreProvisionedVolumeTestDriver
		volumeID   string
		// Set to true if the volume should be deleted automatically after test
		skipManuallyDeletingVolume bool
	)

	ginkgo.BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		testDriver = driver.InitAzureFileCSIDriver()
	})

	ginkgo.AfterEach(func() {
		if !skipManuallyDeletingVolume {
			req := &csi.DeleteVolumeRequest{
				VolumeId: volumeID,
			}
			_, err := azurefileDriver.DeleteVolume(context.Background(), req)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("create volume %q error: %v", volumeID, err))
			}
		}
	})

	ginkgo.It("[env] should use a pre-provisioned volume and mount it as readOnly in a pod", func() {
		req := makeCreateVolumeReq("pre-provisioned-readOnly")
		resp, err := azurefileDriver.CreateVolume(context.Background(), req)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("create volume error: %v", err))
		}
		volumeID = resp.Volume.VolumeId
		ginkgo.By(fmt.Sprintf("Successfully provisioned AzureFile volume: %q\n", volumeID))

		diskSize := fmt.Sprintf("%dGi", defaultDiskSize)
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

	ginkgo.It(fmt.Sprintf("[env] should use a pre-provisioned volume and retain PV with reclaimPolicy %q", v1.PersistentVolumeReclaimRetain), func() {
		req := makeCreateVolumeReq("pre-provisioned-retain-reclaimPolicy")
		resp, err := azurefileDriver.CreateVolume(context.Background(), req)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("create volume error: %v", err))
		}
		volumeID = resp.Volume.VolumeId
		ginkgo.By(fmt.Sprintf("Successfully provisioned AzureFile volume: %q\n", volumeID))

		diskSize := fmt.Sprintf("%dGi", defaultDiskSize)
		reclaimPolicy := v1.PersistentVolumeReclaimRetain
		volumes := []testsuites.VolumeDetails{
			{
				VolumeID:      volumeID,
				FSType:        "ext4",
				ClaimSize:     diskSize,
				ReclaimPolicy: &reclaimPolicy,
			},
		}
		test := testsuites.PreProvisionedReclaimPolicyTest{
			CSIDriver: testDriver,
			Volumes:   volumes,
		}
		test.Run(cs, ns)
	})
})

func makeCreateVolumeReq(volumeName string) *csi.CreateVolumeRequest {
	req := &csi.CreateVolumeRequest{
		Name: volumeName,
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

	return req
}
