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
	"time"

	"github.com/kubernetes-csi/csi-driver-smb/test/e2e/driver"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisionedVolumeCloningTest will provision required StorageClass(es), PVC(s) and Pod(s)
// ClonedVolumeSize optional for when testing for cloned volume with different size to the original volume
type DynamicallyProvisionedVolumeCloningTest struct {
	CSIDriver              driver.DynamicPVTestDriver
	Pod                    PodDetails
	PodWithClonedVolume    PodDetails
	ClonedVolumeSize       string
	StorageClassParameters map[string]string
}

func (t *DynamicallyProvisionedVolumeCloningTest) Run(ctx context.Context, client clientset.Interface, namespace *v1.Namespace) {
	// create the storageClass
	tsc, tscCleanup := t.Pod.Volumes[0].CreateStorageClass(ctx, client, namespace, t.CSIDriver, t.StorageClassParameters)
	defer tscCleanup(ctx)

	// create the pod
	t.Pod.Volumes[0].StorageClass = tsc.storageClass
	tpod, cleanups := t.Pod.SetupWithDynamicVolumes(ctx, client, namespace, t.CSIDriver, t.StorageClassParameters)
	for i := range cleanups {
		defer cleanups[i](ctx)
	}

	ginkgo.By("deploying the pod")
	tpod.Create(ctx)
	defer tpod.Cleanup(ctx)
	ginkgo.By("checking that the pod's command exits with no error")
	tpod.WaitForSuccess(ctx)
	ginkgo.By("sleep 5s and then clone volume")
	time.Sleep(5 * time.Second)

	ginkgo.By("cloning existing volume")
	clonedVolume := t.Pod.Volumes[0]
	clonedVolume.DataSource = &DataSource{
		Name: tpod.pod.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName,
		Kind: VolumePVCKind,
	}
	clonedVolume.StorageClass = tsc.storageClass

	if t.ClonedVolumeSize != "" {
		clonedVolume.ClaimSize = t.ClonedVolumeSize
	}

	t.PodWithClonedVolume.Volumes = []VolumeDetails{clonedVolume}
	tpod, cleanups = t.PodWithClonedVolume.SetupWithDynamicVolumes(ctx, client, namespace, t.CSIDriver, t.StorageClassParameters)
	for i := range cleanups {
		defer cleanups[i](ctx)
	}

	ginkgo.By("deploying a second pod with cloned volume")
	tpod.Create(ctx)
	defer tpod.Cleanup(ctx)
	ginkgo.By("checking that the pod's command exits with no error")
	tpod.WaitForSuccess(ctx)
}
