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
	"context"

	"github.com/kubernetes-csi/csi-driver-smb/test/e2e/driver"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisionedInlineVolumeTest will provision required Source, SecretName
// Waiting for the PV provisioner to create an inline volume
// Testing if the Pod(s) Cmd is run with a 0 exit code
type DynamicallyProvisionedInlineVolumeTest struct {
	CSIDriver  driver.DynamicPVTestDriver
	Pods       []PodDetails
	Source     string
	SecretName string
	ReadOnly   bool
}

func (t *DynamicallyProvisionedInlineVolumeTest) Run(ctx context.Context, client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		tpod, cleanup := pod.SetupWithCSIInlineVolumes(client, namespace, t.Source, t.SecretName, t.ReadOnly)

		// defer must be called here for resources not get removed before using them
		for i := range cleanup {
			defer cleanup[i]()
		}

		ginkgo.By("deploying the pod")
		tpod.Create(ctx)
		defer tpod.Cleanup(ctx)
		ginkgo.By("checking that the pods command exits with no error")
		tpod.WaitForSuccess(ctx)
	}
}
