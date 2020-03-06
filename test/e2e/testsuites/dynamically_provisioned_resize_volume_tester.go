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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	volumehelper "sigs.k8s.io/azurefile-csi-driver/pkg/util"
	"sigs.k8s.io/azurefile-csi-driver/test/e2e/driver"
)

// DynamicallyProvisionedResizeVolumeTest will provision required StorageClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to resize the PV
// Testing if the PV is resized successfully.
type DynamicallyProvisionedResizeVolumeTest struct {
	CSIDriver              driver.DynamicPVTestDriver
	Pods                   []PodDetails
	StorageClassParameters map[string]string
}

func (t *DynamicallyProvisionedResizeVolumeTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	for _, pod := range t.Pods {
		tpod, cleanup := pod.SetupWithDynamicVolumes(client, namespace, t.CSIDriver, t.StorageClassParameters)
		for i := range cleanup {
			defer cleanup[i]()
		}

		ginkgo.By("deploying the pod")
		tpod.Create()
		defer tpod.Cleanup()
		ginkgo.By("checking that the pods command exits with no error")
		tpod.WaitForSuccess()

		pvcName := tpod.pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName
		pvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to get original pvc(%s): %v", pvcName, err))
		}

		originalSize := pvc.Spec.Resources.Requests["storage"]
		delta := resource.Quantity{}
		delta.Set(volumehelper.GiBToBytes(1))
		originalSize.Add(delta)
		pvc.Spec.Resources.Requests["storage"] = originalSize

		ginkgo.By("resizing the pvc")
		updatedPvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Update(pvc)
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to resize pvc(%s): %v", pvcName, err))
		}
		updatedSize := updatedPvc.Spec.Resources.Requests["storage"]

		ginkgo.By("checking the resizing result")
		newPvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(tpod.pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to get new pvc(%s): %v", pvcName, err))
		}
		newSize := newPvc.Spec.Resources.Requests["storage"]
		if !newSize.Equal(updatedSize) {
			framework.Failf("newSize(%+v) is not equal to updatedSize(%+v)", newSize, updatedSize)
		}
	}
}
