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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/azurefile-csi-driver/pkg/azurefile"
	"sigs.k8s.io/azurefile-csi-driver/test/utils/azure"
	"sigs.k8s.io/azurefile-csi-driver/test/utils/credentials"
	"sigs.k8s.io/azurefile-csi-driver/test/utils/testutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
)

const kubeconfigEnvVar = "KUBECONFIG"

var azurefileDriver *azurefile.Driver

func init() {
	// k8s.io/kubernetes/test/e2e/framework requires env KUBECONFIG to be set
	// it does not fall back to defaults
	if os.Getenv(kubeconfigEnvVar) == "" {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		os.Setenv(kubeconfigEnvVar, kubeconfig)
	}
	framework.HandleFlags()
	framework.AfterReadingAllFlags(&framework.TestContext)
}

var _ = BeforeSuite(func() {
	creds, err := credentials.CreateAzureCredentialFile(false)
	Expect(err).NotTo(HaveOccurred())
	azureClient, err := azure.GetAzureClient(creds.Cloud, creds.SubscriptionID, creds.AADClientID, creds.TenantID, creds.AADClientSecret)
	Expect(err).NotTo(HaveOccurred())
	_, err = azureClient.EnsureResourceGroup(context.Background(), creds.ResourceGroup, creds.Location, nil)
	Expect(err).NotTo(HaveOccurred())

	// Need to login to ACR using SP credential if we are running in Prow so we can push test images.
	// If running locally, user should run 'docker login' before running E2E tests
	if testutil.IsRunningInProw() {
		registry := os.Getenv("REGISTRY")
		Expect(registry).NotTo(Equal(""))

		log.Println("Attempting docker login with Azure service principal")
		cmd := exec.Command("docker", "login", fmt.Sprintf("--username=%s", creds.AADClientID), fmt.Sprintf("--password=%s", creds.AADClientSecret), registry)
		err := cmd.Run()
		Expect(err).NotTo(HaveOccurred())
		log.Println("docker login is successful")
	}

	// Install Azure File CSI Driver on cluster from project root
	err = os.Chdir("../..")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := os.Chdir("test/e2e")
		Expect(err).NotTo(HaveOccurred())
	}()

	projectRoot, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	Expect(strings.HasSuffix(projectRoot, "azurefile-csi-driver")).To(Equal(true))

	log.Println("Installing Azure File CSI Driver...")
	cmd := exec.Command("make", "e2e-bootstrap")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	log.Println("Azure File CSI Driver installed")

	nodeid := os.Getenv("nodeid")
	azurefileDriver = azurefile.NewDriver(nodeid)
	go func() {
		os.Setenv("AZURE_CREDENTIAL_FILE", credentials.TempAzureCredentialFilePath)
		azurefileDriver.Run(fmt.Sprintf("unix:///tmp/csi-%s.sock", uuid.NewUUID().String()))
	}()
})

var _ = AfterSuite(func() {
	err := os.Chdir("../..")
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		err := os.Chdir("test/e2e")
		Expect(err).NotTo(HaveOccurred())
	}()

	projectRoot, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	Expect(strings.HasSuffix(projectRoot, "azurefile-csi-driver")).To(Equal(true))

	log.Println("Uninstalling Azure File CSI Driver...")
	cmd := exec.Command("make", "e2e-teardown")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	log.Println("Azure File CSI Driver uninstalled")

	err = credentials.DeleteAzureCredentialFile()
	Expect(err).NotTo(HaveOccurred())
})

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AzureFile CSI Driver End-to-End Tests")
}
