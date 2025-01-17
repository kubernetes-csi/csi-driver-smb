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

package e2e

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubernetes-csi/csi-driver-smb/pkg/smb"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	"github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
)

const (
	kubeconfigEnvVar             = "KUBECONFIG"
	reportDirEnv                 = "ARTIFACTS"
	testWindowsEnvVar            = "TEST_WINDOWS"
	testWinServerVerEnvVar       = "WINDOWS_SERVER_VERSION"
	preInstallDriverEnvVar       = "PRE_INSTALL_SMB_PROVISIONER"
	defaultReportDir             = "test/e2e"
	testSmbSourceEnvVar          = "TEST_SMB_SOURCE"
	testSmbSecretNameEnvVar      = "TEST_SMB_SECRET_NAME"
	testSmbSecretNamespaceEnvVar = "TEST_SMB_SECRET_NAMESPACE"
	defaultSmbSource             = "//smb-server.default.svc.cluster.local/share"
	defaultSmbSecretName         = "smbcreds"
	defaultSmbSecretNamespace    = "default"
	accountNameForTest           = "YW5keXNzZGZpbGUK"
)

var (
	smbDriver                      *smb.Driver
	isWindowsCluster               = os.Getenv(testWindowsEnvVar) != ""
	isWindowsHostProcessDeployment = os.Getenv("WINDOWS_USE_HOST_PROCESS_CONTAINERS") != ""
	winServerVer                   = os.Getenv(testWinServerVerEnvVar)
	preInstallDriver               = os.Getenv(preInstallDriverEnvVar) == "true"
	defaultStorageClassParameters  = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"csi.storage.k8s.io/provisioner-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/provisioner-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"csi.storage.k8s.io/node-stage-secret-name":       getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace":  getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
	}
	subDirStorageClassParameters = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"subDir": "${pvc.metadata.namespace}/${pvc.metadata.name}",
		"csi.storage.k8s.io/provisioner-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/provisioner-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"csi.storage.k8s.io/node-stage-secret-name":       getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace":  getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
	}
	retainStorageClassParameters = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"csi.storage.k8s.io/provisioner-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/provisioner-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"csi.storage.k8s.io/node-stage-secret-name":       getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace":  getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"onDelete": "retain",
	}
	archiveStorageClassParameters = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"csi.storage.k8s.io/provisioner-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/provisioner-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"csi.storage.k8s.io/node-stage-secret-name":       getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace":  getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"onDelete": "archive",
	}
	archiveSubDirStorageClassParameters = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"subDir": "${pvc.metadata.namespace}/${pvc.metadata.name}",
		"csi.storage.k8s.io/provisioner-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/provisioner-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"csi.storage.k8s.io/node-stage-secret-name":       getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace":  getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
		"onDelete": "archive",
	}
	noProvisionerSecretStorageClassParameters = map[string]string{
		"source": getSmbTestEnvVarValue(testSmbSourceEnvVar, defaultSmbSource),
		"csi.storage.k8s.io/node-stage-secret-name":      getSmbTestEnvVarValue(testSmbSecretNameEnvVar, defaultSmbSecretName),
		"csi.storage.k8s.io/node-stage-secret-namespace": getSmbTestEnvVarValue(testSmbSecretNamespaceEnvVar, defaultSmbSecretNamespace),
	}
)

type testCmd struct {
	command  string
	args     []string
	startLog string
	endLog   string
}

var _ = ginkgo.BeforeSuite(func() {
	// k8s.io/kubernetes/test/e2e/framework requires env KUBECONFIG to be set
	// it does not fall back to defaults
	if os.Getenv(kubeconfigEnvVar) == "" {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		os.Setenv(kubeconfigEnvVar, kubeconfig)
	}
	handleFlags()
	framework.AfterReadingAllFlags(&framework.TestContext)

	kubeconfig := os.Getenv(kubeconfigEnvVar)
	log.Println(testWinServerVerEnvVar, os.Getenv(testWinServerVerEnvVar), fmt.Sprintf("%v", winServerVer))

	// Install SMB provisioner on cluster
	installSMBProvisioner := testCmd{
		command:  "make",
		args:     []string{"install-smb-provisioner"},
		startLog: "Installing SMB provisioner...",
		endLog:   "SMB provisioner installed",
	}
	// Install SMB CSI Driver on cluster from project root
	e2eBootstrap := testCmd{
		command:  "make",
		args:     []string{"e2e-bootstrap"},
		startLog: "Installing SMB CSI Driver...",
		endLog:   "SMB CSI Driver installed",
	}

	createMetricsSVC := testCmd{
		command:  "make",
		args:     []string{"create-metrics-svc"},
		startLog: "create metrics service ...",
		endLog:   "metrics service created",
	}
	if !preInstallDriver {
		execTestCmd([]testCmd{installSMBProvisioner, e2eBootstrap, createMetricsSVC})
	}

	nodeid := os.Getenv("nodeid")
	options := smb.DriverOptions{
		NodeID:               nodeid,
		DriverName:           smb.DefaultDriverName,
		EnableGetVolumeStats: false,
	}

	smbDriver = smb.NewDriver(&options)
	go func() {
		smbDriver.Run(fmt.Sprintf("unix:///tmp/csi-%s.sock", uuid.NewUUID().String()), kubeconfig, false)
	}()

	var source string
	if isWindowsCluster {
		err := os.Chdir("../..")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		defer func() {
			err := os.Chdir("test/e2e")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}()
	}

	if isWindowsHostProcessDeployment {
		decodedBytes, err := base64.StdEncoding.DecodeString(accountNameForTest)
		if err != nil {
			log.Printf("Error decoding base64 string: %v\n", err)
			return
		}
		source = fmt.Sprintf("//%s.file.core.windows.net/test", strings.TrimRight(string(decodedBytes), "\n"))

		createSMBCredsScript := "test/utils/create_smbcreds_windows.sh"
		log.Printf("run script: %s\n", createSMBCredsScript)

		cmd := exec.Command("bash", createSMBCredsScript)
		output, err := cmd.CombinedOutput()
		log.Printf("got output: %v, error: %v\n", string(output), err)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	} else if isWindowsCluster {
		getSMBPublicIPScript := "test/utils/get_smb_svc_public_ip.sh"
		log.Printf("run script: %s\n", getSMBPublicIPScript)

		cmd := exec.Command("bash", getSMBPublicIPScript)
		output, err := cmd.CombinedOutput()
		log.Printf("got output: %v, error: %v\n", string(output), err)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		smbPublicIP := strings.TrimSuffix(string(output), "\n")
		source = fmt.Sprintf("//%s/share", smbPublicIP)
	}

	if isWindowsCluster {
		log.Printf("use source on Windows: %v\n", source)
		defaultStorageClassParameters["source"] = source
		retainStorageClassParameters["source"] = source
		archiveStorageClassParameters["source"] = source
		archiveSubDirStorageClassParameters["source"] = source
		subDirStorageClassParameters["source"] = source
		noProvisionerSecretStorageClassParameters["source"] = source
	}
})

var _ = ginkgo.AfterSuite(func() {
	if !isWindowsCluster {
		createExampleDeployment := testCmd{
			command:  "bash",
			args:     []string{"hack/verify-examples.sh"},
			startLog: "create example deployments",
			endLog:   "example deployments created",
		}
		if !preInstallDriver {
			execTestCmd([]testCmd{createExampleDeployment})
		}
	}

	smbLog := testCmd{
		command:  "bash",
		args:     []string{"test/utils/smb_log.sh"},
		startLog: "===================smb log===================",
		endLog:   "===================================================",
	}
	e2eTeardown := testCmd{
		command:  "make",
		args:     []string{"e2e-teardown"},
		startLog: "Uninstalling SMB CSI Driver...",
		endLog:   "SMB Driver uninstalled",
	}
	e2eTeardownCmds := []testCmd{smbLog}
	if !preInstallDriver {
		e2eTeardownCmds = append(e2eTeardownCmds, e2eTeardown)
	}
	execTestCmd(e2eTeardownCmds)

	// install/uninstall CSI Driver deployment scripts test
	installDriver := testCmd{
		command:  "bash",
		args:     []string{"deploy/install-driver.sh", "master", "local"},
		startLog: "===================install CSI Driver deployment scripts test===================",
		endLog:   "===================================================",
	}
	uninstallDriver := testCmd{
		command:  "bash",
		args:     []string{"deploy/uninstall-driver.sh", "master", "local"},
		startLog: "===================uninstall CSI Driver deployment scripts test===================",
		endLog:   "===================================================",
	}
	if !preInstallDriver {
		execTestCmd([]testCmd{installDriver, uninstallDriver})
	}
})

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	reportDir := os.Getenv(reportDirEnv)
	if reportDir == "" {
		reportDir = defaultReportDir
	}
	r := []ginkgo.Reporter{reporters.NewJUnitReporter(path.Join(reportDir, "junit_01.xml"))}
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "SMB CSI Driver End-to-End Tests", r)
}

func execTestCmd(cmds []testCmd) {
	err := os.Chdir("../..")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer func() {
		err := os.Chdir("test/e2e")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}()

	projectRoot, err := os.Getwd()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(strings.HasSuffix(projectRoot, "csi-driver-smb")).To(gomega.Equal(true))

	for _, cmd := range cmds {
		log.Println(cmd.startLog)
		cmdSh := exec.Command(cmd.command, cmd.args...)
		cmdSh.Dir = projectRoot
		cmdSh.Stdout = os.Stdout
		cmdSh.Stderr = os.Stderr
		err = cmdSh.Run()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		log.Println(cmd.endLog)
	}
}

func convertToPowershellCommandIfNecessary(command string) string {
	if !isWindowsCluster {
		return command
	}

	switch command {
	case "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data":
		return "echo 'hello world' | Out-File -FilePath C:\\mnt\\test-1\\data.txt; Get-Content C:\\mnt\\test-1\\data.txt | findstr 'hello world'"
	case "touch /mnt/test-1/data":
		return "echo $null >> C:\\mnt\\test-1\\data"
	case "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 100; done":
		return "while (1) { Add-Content -Encoding Unicode C:\\mnt\\test-1\\data.txt $(Get-Date -Format u); sleep 1 }"
	case "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 100; done":
		return "Add-Content -Encoding Unicode C:\\mnt\\test-1\\data.txt 'hello world'; while (1) { sleep 1 }"
	case "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done":
		return "Add-Content -Encoding Unicode C:\\mnt\\test-1\\data.txt 'hello world'; while (1) { sleep 1 }"
	}

	return command
}

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	framework.RegisterClusterFlags(flag.CommandLine)
	flag.Parse()
}

func skipIfTestingInWindowsCluster() {
	if isWindowsCluster {
		ginkgo.Skip("test case not supported by Windows clusters")
	}
}

// getSmbTestEnvVarValue gets the smbTestEnvValue from env var if the var does not set use the defaultVarValue
func getSmbTestEnvVarValue(envVarName string, defaultVarValue string) (smbTestEnvValue string) {
	smbTestEnvValue = os.Getenv(envVarName)
	if smbTestEnvValue == "" {
		smbTestEnvValue = defaultVarValue
	}
	return
}
