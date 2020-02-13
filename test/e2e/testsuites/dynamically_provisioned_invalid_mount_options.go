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

package testsuites

import (
	"sigs.k8s.io/azurefile-csi-driver/test/e2e/driver"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisionedInvalidMountOptions will provision a storage class with invalid mount options
// Testing if the Pod(s) Cmd is run with a 0 exit code
type DynamicallyProvisionedInvalidMountOptions struct {
	CSIDriver driver.DynamicPVTestDriver
	Pods      []PodDetails
}

func (t *DynamicallyProvisionedInvalidMountOptions) Run(client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		tpod, cleanup := pod.SetupWithDynamicVolumes(client, namespace, t.CSIDriver)
		// defer must be called here for resources not get removed before using them
		for i := range cleanup {
			defer cleanup[i]()
		}

		ginkgo.By("deploying the pod")
		tpod.Create()
		defer tpod.Cleanup()
		ginkgo.By("checking that the pod has 'FailedMount' event")
		tpod.WaitForFailedMountError()
	}
}
