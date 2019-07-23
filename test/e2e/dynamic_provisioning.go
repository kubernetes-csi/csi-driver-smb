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
	"fmt"
	"os"

	"github.com/kubernetes-sigs/azurefile-csi-driver/pkg/azurefile"
	"github.com/kubernetes-sigs/azurefile-csi-driver/test/e2e/driver"
	"github.com/kubernetes-sigs/azurefile-csi-driver/test/e2e/testsuites"
	. "github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("azurefile")

	var (
		cs         clientset.Interface
		ns         *v1.Namespace
		testDriver driver.PVTestDriver
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
	})

	testDriver = driver.InitAzureFileCSIDriver()
	It(fmt.Sprintf("should create a volume on demand"), func() {
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						ClaimSize: "10Gi",
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create multiple PV objects, bind to PVCs and attach all to different pods on the same node", func() {
		pods := []testsuites.PodDetails{
			{
				Cmd: "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 1; done",
				Volumes: []testsuites.VolumeDetails{
					{
						FSType:    "ext3",
						ClaimSize: "10Gi",
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
			{
				Cmd: "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 1; done",
				Volumes: []testsuites.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCollocatedPodTest{
			CSIDriver:    testDriver,
			Pods:         pods,
			ColocatePods: true,
		}
		test.Run(cs, ns)
	})

	// Track issue https://github.com/kubernetes/kubernetes/issues/70505
	It("should create a volume on demand and mount it as readOnly in a pod", func() {
		pods := []testsuites.PodDetails{
			{
				Cmd: "touch /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
							ReadOnly:          true,
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedReadOnlyVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})

	It("should create a deployment object, write and read to it, delete the pod and write and read to it again", func() {
		pod := testsuites.PodDetails{
			Cmd: "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 1; done",
			Volumes: []testsuites.VolumeDetails{
				{
					FSType:    "ext3",
					ClaimSize: "10Gi",
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedDeletePodTest{
			CSIDriver: testDriver,
			Pod:       pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            []string{"cat", "/mnt/test-1/data"},
				ExpectedString: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns)
	})

	It(fmt.Sprintf("should delete PV with reclaimPolicy %q", v1.PersistentVolumeReclaimDelete), func() {
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		volumes := []testsuites.VolumeDetails{
			{
				FSType:        "ext4",
				ClaimSize:     "10Gi",
				ReclaimPolicy: &reclaimPolicy,
			},
		}
		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver: testDriver,
			Volumes:   volumes,
		}
		test.Run(cs, ns)
	})

	It(fmt.Sprintf("[env] should retain PV with reclaimPolicy %q", v1.PersistentVolumeReclaimRetain), func() {
		reclaimPolicy := v1.PersistentVolumeReclaimRetain
		volumes := []testsuites.VolumeDetails{
			{
				FSType:        "ext4",
				ClaimSize:     "10Gi",
				ReclaimPolicy: &reclaimPolicy,
			},
		}
		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver: testDriver,
			Volumes:   volumes,
			Azurefile: azurefileDriver,
		}
		test.Run(cs, ns)
	})
})
