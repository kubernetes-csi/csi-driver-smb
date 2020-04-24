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

// PreProvisionedExistingCredentialsTest will provision required StorageClass(es), PVC(s) and Pod(s)
// Testing that the Pod(s) can be created successfully with existing credentials in k8s cluster
type PreProvisionedExistingCredentialsTest struct {
	CSIDriver driver.PreProvisionedVolumeTestDriver
	Pods      []PodDetails
	Azurefile *azurefile.Driver
}

func (t *PreProvisionedExistingCredentialsTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		for n, volume := range pod.Volumes {
			resourceGroupName, accountName, _, fileShareName, _, err := t.Azurefile.GetAccountInfo(volume.VolumeID, nil, nil)
			if err != nil {
				framework.ExpectNoError(err, fmt.Sprintf("Error GetContainerInfo from volumeID(%s): %v", volume.VolumeID, err))
				return
			}
			parameters := map[string]string{
				"resourceGroup":  resourceGroupName,
				"storageAccount": accountName,
				"shareName":      fileShareName,
			}

			ginkgo.By("creating the storageclass with existing credentials")
			sc := t.CSIDriver.GetPreProvisionStorageClass(parameters, volume.MountOptions, volume.ReclaimPolicy, volume.VolumeBindingMode, volume.AllowedTopologyValues, namespace.Name)
			tsc := NewTestStorageClass(client, namespace, sc)
			createdStorageClass := tsc.Create()
			defer tsc.Cleanup()

			ginkgo.By("creating pvc with storageclass")
			tpvc := NewTestPersistentVolumeClaim(client, namespace, volume.ClaimSize, volume.VolumeMode, &createdStorageClass)
			tpvc.Create()
			defer tpvc.Cleanup()

			ginkgo.By("validating the pvc")
			tpvc.WaitForBound()
			tpvc.ValidateProvisionedPersistentVolume()

			tpod := NewTestPod(client, namespace, pod.Cmd, pod.IsWindows)
			tpod.SetupVolume(tpvc.persistentVolumeClaim, fmt.Sprintf("%s%d", volume.VolumeMount.NameGenerate, n+1), fmt.Sprintf("%s%d", volume.VolumeMount.MountPathGenerate, n+1), volume.VolumeMount.ReadOnly)
			ginkgo.By("deploying the pod")
			tpod.Create()
			defer tpod.Cleanup()
			ginkgo.By("checking that the pods command exits with no error")
			tpod.WaitForSuccess()
		}
	}
}
