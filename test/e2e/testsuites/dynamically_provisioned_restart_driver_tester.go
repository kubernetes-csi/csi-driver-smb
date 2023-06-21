/*
Copyright 2021 The Kubernetes Authors.

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

// DynamicallyProvisionedRestartDriverTest will test to ensure that restarting driver doesn't affect pod mounting.
// It will mount a pod, restart the driver daemonset and ensure that the pod still has access to original volume.
type DynamicallyProvisionedRestartDriverTest struct {
	CSIDriver              driver.DynamicPVTestDriver
	Pod                    PodDetails
	PodCheck               *PodExecCheck
	StorageClassParameters map[string]string
	RestartDriverFunc      func()
}

func (t *DynamicallyProvisionedRestartDriverTest) Run(ctx context.Context, client clientset.Interface, namespace *v1.Namespace) {
	tDeployment, cleanup := t.Pod.SetupDeployment(ctx, client, namespace, t.CSIDriver, t.StorageClassParameters)
	// defer must be called here for resources not get removed before using them
	for i := range cleanup {
		defer cleanup[i](ctx)
	}

	ginkgo.By("creating the deployment for the pod")
	tDeployment.Create(ctx)

	ginkgo.By("checking that the pod is running")
	tDeployment.WaitForPodReady(ctx)

	if t.PodCheck != nil {
		ginkgo.By("checking if pod is able to access volume")
		tDeployment.PollForStringInPodsExec(t.PodCheck.Cmd, t.PodCheck.ExpectedString)
	}

	// restart the driver
	ginkgo.By("restarting the driver daemonset")
	t.RestartDriverFunc()

	// check if original pod could still access volume
	if t.PodCheck != nil {
		ginkgo.By("checking if pod still has access to volume after driver restart")
		tDeployment.PollForStringInPodsExec(t.PodCheck.Cmd, t.PodCheck.ExpectedString)
	}
}
