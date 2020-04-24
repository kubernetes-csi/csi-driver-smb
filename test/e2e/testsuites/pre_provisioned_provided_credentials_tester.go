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

package testsuites

import (
	"fmt"

	"github.com/onsi/ginkgo"

	"sigs.k8s.io/azurefile-csi-driver/pkg/azurefile"
	"sigs.k8s.io/azurefile-csi-driver/test/e2e/driver"

	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

// PreProvisionedProvidedCredentiasTest will provision required PV(s), PVC(s) and Pod(s)
// Testing that the Pod(s) can be created successfully with provided storage account name and key(or sastoken)
type PreProvisionedProvidedCredentiasTest struct {
	CSIDriver driver.PreProvisionedVolumeTestDriver
	Pods      []PodDetails
	Azurefile *azurefile.Driver
}

func (t *PreProvisionedProvidedCredentiasTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		for n, volume := range pod.Volumes {
			_, accountName, accountKey, fileShareName, _, err := t.Azurefile.GetAccountInfo(volume.VolumeID, nil, nil)
			framework.ExpectNoError(err, fmt.Sprintf("Error GetAccountInfo from volumeID(%s): %v", volume.VolumeID, err))

			ginkgo.By("creating the secret")
			secreteData := map[string]string{"azurestorageaccountname": accountName}
			secreteData["azurestorageaccountkey"] = accountKey
			tsecret := NewTestSecret(client, namespace, volume.NodeStageSecretRef, secreteData)
			tsecret.Create()
			defer tsecret.Cleanup()

			pod.Volumes[n].ShareName = fileShareName
			tpod, cleanup := pod.SetupWithPreProvisionedVolumes(client, namespace, t.CSIDriver)
			// defer must be called here for resources not get removed before using them
			for i := range cleanup {
				defer cleanup[i]()
			}

			ginkgo.By("deploying the pod")
			tpod.Create()
			defer tpod.Cleanup()
			ginkgo.By("checking that the pods command exits with no error")
			tpod.WaitForSuccess()
		}
	}
}
