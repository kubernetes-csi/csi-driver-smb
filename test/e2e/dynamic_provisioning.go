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
		cs              clientset.Interface
		ns              *v1.Namespace
		azureFileDriver driver.PVTestDriver
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
	})

	azureFileDriver = driver.InitAzureFileCSIDriver()
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
			CSIDriver: azureFileDriver,
			Pods:      pods,
		}
		test.Run(cs, ns)
	})
})
