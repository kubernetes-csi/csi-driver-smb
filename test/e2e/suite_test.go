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
	"github.com/kubernetes-csi/csi-driver-smb/test/utils/testutil"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
)

const (
	kubeconfigEnvVar  = "KUBECONFIG"
	reportDirEnv      = "ARTIFACTS"
	testWindowsEnvVar = "TEST_WINDOWS"
	defaultReportDir  = "test/e2e"
)

var (
	smbDriver                     *smb.Driver
	isWindowsCluster              = os.Getenv(testWindowsEnvVar) != ""
	defaultStorageClassParameters = map[string]string{
		"source": "//smb-server.default.svc.cluster.local/share",
		"csi.storage.k8s.io/node-stage-secret-name":      "smbcreds",
		"csi.storage.k8s.io/node-stage-secret-namespace": "default",
		"createSubDir": "false",
	}
	storageClassCreateSubDir = map[string]string{
		"source": "//smb-server.default.svc.cluster.local/share",
		"csi.storage.k8s.io/node-stage-secret-name":       "smbcreds",
		"csi.storage.k8s.io/node-stage-secret-namespace":  "default",
		"csi.storage.k8s.io/provisioner-secret-name":      "smbcreds",
		"csi.storage.k8s.io/provisioner-secret-namespace": "default",
		"createSubDir": "true",
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

	if testutil.IsRunningInProw() {
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

		execTestCmd([]testCmd{installSMBProvisioner, e2eBootstrap, createMetricsSVC})

		nodeid := os.Getenv("nodeid")
		kubeconfig := os.Getenv(kubeconfigEnvVar)
		smbDriver = smb.NewDriver(nodeid, smb.DefaultDriverName)
		go func() {
			smbDriver.Run(fmt.Sprintf("unix:///tmp/csi-%s.sock", uuid.NewUUID().String()), kubeconfig, false)
		}()
	}

	if isWindowsCluster {
		err := os.Chdir("../..")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		defer func() {
			err := os.Chdir("test/e2e")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}()

		getSMBPublicIPScript := "test/utils/get_smb_svc_public_ip.sh"
		log.Printf("run script: %s\n", getSMBPublicIPScript)

		cmd := exec.Command("bash", getSMBPublicIPScript)
		output, err := cmd.CombinedOutput()
		log.Printf("got output: %v, error: %v\n", string(output), err)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		smbPublicIP := strings.TrimSuffix(string(output), "\n")
		source := `//` + smbPublicIP + `/share`

		log.Printf("use source on Windows: %v\n", source)
		defaultStorageClassParameters["source"] = source
		storageClassCreateSubDir["source"] = source
	}
})

var _ = ginkgo.AfterSuite(func() {
	if testutil.IsRunningInProw() {
		if !isWindowsCluster {
			createExampleDeployment := testCmd{
				command:  "bash",
				args:     []string{"hack/verify-examples.sh"},
				startLog: "create example deployments",
				endLog:   "example deployments created",
			}
			execTestCmd([]testCmd{createExampleDeployment})
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
		execTestCmd([]testCmd{smbLog, e2eTeardown})

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
