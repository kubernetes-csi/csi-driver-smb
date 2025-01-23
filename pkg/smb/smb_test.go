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

package smb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	fakeNodeID = "fakeNodeID"
)

func NewFakeDriver() *Driver {
	options := DriverOptions{
		NodeID:               fakeNodeID,
		DriverName:           DefaultDriverName,
		EnableGetVolumeStats: true,
	}
	return NewDriver(&options)
}

func TestNewFakeDriver(t *testing.T) {
	options := DriverOptions{
		NodeID:               fakeNodeID,
		DriverName:           DefaultDriverName,
		EnableGetVolumeStats: true,
	}
	d := NewDriver(&options)
	assert.NotNil(t, d)
}

func TestIsCorruptedDir(t *testing.T) {
	existingMountPath, err := os.MkdirTemp(os.TempDir(), "csi-mount-test")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(existingMountPath)

	curruptedPath := filepath.Join(existingMountPath, "curruptedPath")
	if err := os.Symlink(existingMountPath, curruptedPath); err != nil {
		t.Fatalf("failed to create curruptedPath: %v", err)
	}

	tests := []struct {
		desc           string
		dir            string
		expectedResult bool
	}{
		{
			desc:           "NotExist dir",
			dir:            "/tmp/NotExist",
			expectedResult: false,
		},
		{
			desc:           "Existing dir",
			dir:            existingMountPath,
			expectedResult: false,
		},
	}

	for i, test := range tests {
		isCorruptedDir := IsCorruptedDir(test.dir)
		assert.Equal(t, test.expectedResult, isCorruptedDir, "TestCase[%d]: %s", i, test.desc)
	}
}

func TestRun(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Successful run",
			testFunc: func(_ *testing.T) {
				d := NewFakeDriver()
				d.Run("tcp://127.0.0.1:0", "", true)
			},
		},
		{
			name: "Successful run with node ID missing",
			testFunc: func(_ *testing.T) {
				d := NewFakeDriver()
				d.NodeID = ""
				d.Run("tcp://127.0.0.1:0", "", true)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testFunc)
	}
}

func TestGetMountOptions(t *testing.T) {
	tests := []struct {
		desc    string
		context map[string]string
		result  string
	}{
		{
			desc:    "nil context",
			context: nil,
			result:  "",
		},
		{
			desc:    "empty context",
			context: map[string]string{},
			result:  "",
		},
		{
			desc:    "valid mountOptions",
			context: map[string]string{"mountOptions": "dir_mode=0777"},
			result:  "dir_mode=0777",
		},
		{
			desc:    "valid mountOptions(lowercase)",
			context: map[string]string{"mountoptions": "dir_mode=0777,file_mode=0777,uid=0,gid=0,mfsymlinks"},
			result:  "dir_mode=0777,file_mode=0777,uid=0,gid=0,mfsymlinks",
		},
	}

	for _, test := range tests {
		result := getMountOptions(test.context)
		if result != test.result {
			t.Errorf("Unexpected result: %s, expected: %s", result, test.result)
		}
	}
}

func TestHasGuestMountOptions(t *testing.T) {
	tests := []struct {
		desc    string
		options []string
		result  bool
	}{
		{
			desc:   "empty options",
			result: false,
		},
		{
			desc:    "no guest option",
			options: []string{"a", "b"},
			result:  false,
		},
		{
			desc:    "has guest option",
			options: []string{"a", "b", "guest"},
			result:  true,
		},
	}

	for _, test := range tests {
		result := hasGuestMountOptions(test.options)
		if result != test.result {
			t.Errorf("test(%s): unexpected result: %v, expected: %v", test.desc, result, test.result)
		}
	}
}

func TestSetKeyValueInMap(t *testing.T) {
	tests := []struct {
		desc     string
		m        map[string]string
		key      string
		value    string
		expected map[string]string
	}{
		{
			desc:  "nil map",
			key:   "key",
			value: "value",
		},
		{
			desc:     "empty map",
			m:        map[string]string{},
			key:      "key",
			value:    "value",
			expected: map[string]string{"key": "value"},
		},
		{
			desc:  "non-empty map",
			m:     map[string]string{"k": "v"},
			key:   "key",
			value: "value",
			expected: map[string]string{
				"k":   "v",
				"key": "value",
			},
		},
		{
			desc:     "same key already exists",
			m:        map[string]string{"subDir": "value2"},
			key:      "subDir",
			value:    "value",
			expected: map[string]string{"subDir": "value"},
		},
		{
			desc:     "case insensitive key already exists",
			m:        map[string]string{"subDir": "value2"},
			key:      "subdir",
			value:    "value",
			expected: map[string]string{"subDir": "value"},
		},
	}

	for _, test := range tests {
		setKeyValueInMap(test.m, test.key, test.value)
		if !reflect.DeepEqual(test.m, test.expected) {
			t.Errorf("test[%s]: unexpected output: %v, expected result: %v", test.desc, test.m, test.expected)
		}
	}
}

func TestReplaceWithMap(t *testing.T) {
	tests := []struct {
		desc     string
		str      string
		m        map[string]string
		expected string
	}{
		{
			desc:     "empty string",
			str:      "",
			expected: "",
		},
		{
			desc:     "empty map",
			str:      "",
			m:        map[string]string{},
			expected: "",
		},
		{
			desc:     "empty key",
			str:      "prefix-" + pvNameMetadata,
			m:        map[string]string{"": "pv"},
			expected: "prefix-" + pvNameMetadata,
		},
		{
			desc:     "empty value",
			str:      "prefix-" + pvNameMetadata,
			m:        map[string]string{pvNameMetadata: ""},
			expected: "prefix-",
		},
		{
			desc:     "one replacement",
			str:      "prefix-" + pvNameMetadata,
			m:        map[string]string{pvNameMetadata: "pv"},
			expected: "prefix-pv",
		},
		{
			desc:     "multiple replacements",
			str:      pvcNamespaceMetadata + pvcNameMetadata,
			m:        map[string]string{pvcNamespaceMetadata: "namespace", pvcNameMetadata: "pvcname"},
			expected: "namespacepvcname",
		},
	}

	for _, test := range tests {
		result := replaceWithMap(test.str, test.m)
		if result != test.expected {
			t.Errorf("test[%s]: unexpected output: %v, expected result: %v", test.desc, result, test.expected)
		}
	}
}

func TestValidateOnDeleteValue(t *testing.T) {
	tests := []struct {
		desc     string
		onDelete string
		expected error
	}{
		{
			desc:     "empty value",
			onDelete: "",
			expected: nil,
		},
		{
			desc:     "delete value",
			onDelete: "delete",
			expected: nil,
		},
		{
			desc:     "retain value",
			onDelete: "retain",
			expected: nil,
		},
		{
			desc:     "Retain value",
			onDelete: "Retain",
			expected: nil,
		},
		{
			desc:     "Delete value",
			onDelete: "Delete",
			expected: nil,
		},
		{
			desc:     "Archive value",
			onDelete: "Archive",
			expected: nil,
		},
		{
			desc:     "archive value",
			onDelete: "archive",
			expected: nil,
		},
		{
			desc:     "invalid value",
			onDelete: "invalid",
			expected: fmt.Errorf("invalid value %s for OnDelete, supported values are %v", "invalid", supportedOnDeleteValues),
		},
	}

	for _, test := range tests {
		result := validateOnDeleteValue(test.onDelete)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("test[%s]: unexpected output: %v, expected result: %v", test.desc, result, test.expected)
		}
	}
}

func TestAppendMountOptions(t *testing.T) {
	tests := []struct {
		desc     string
		options  []string
		newOpts  map[string]string
		expected []string
	}{
		{
			desc:     "empty options",
			options:  nil,
			newOpts:  map[string]string{},
			expected: nil,
		},
		{
			desc:     "empty newOpts",
			options:  []string{"a", "b"},
			newOpts:  map[string]string{},
			expected: []string{"a", "b"},
		},
		{
			desc:     "empty newOpts",
			options:  []string{"a", "b"},
			newOpts:  map[string]string{"c": "d"},
			expected: []string{"a", "b", "c=d"},
		},
		{
			desc:     "duplicate newOpts",
			options:  []string{"a", "b", "c=d"},
			newOpts:  map[string]string{"c": "d"},
			expected: []string{"a", "b", "c=d"},
		},
	}

	for _, test := range tests {
		result := appendMountOptions(test.options, test.newOpts)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("test[%s]: unexpected output: %v, expected result: %v", test.desc, result, test.expected)
		}
	}
}

func TestGetRootPath(t *testing.T) {
	tests := []struct {
		desc     string
		dir      string
		expected string
	}{
		{
			desc:     "empty path",
			dir:      "",
			expected: "",
		},
		{
			desc:     "root path",
			dir:      "/",
			expected: "",
		},
		{
			desc:     "subdir path",
			dir:      "/subdir",
			expected: "",
		},
		{
			desc:     "subdir path without leading slash",
			dir:      "subdir",
			expected: "subdir",
		},
		{
			desc:     "multiple subdir path without leading slash",
			dir:      "subdir/subdir2",
			expected: "subdir",
		},
	}

	for _, test := range tests {
		result := getRootDir(test.dir)
		if result != test.expected {
			t.Errorf("Unexpected result: %s, expected: %s", result, test.expected)
		}
	}
}

func TestGetKubeConfig(t *testing.T) {
	// skip for now as this is very flaky on Windows
	//skipIfTestingOnWindows(t)
	emptyKubeConfig := "empty-Kube-Config"
	validKubeConfig := "valid-Kube-Config"
	fakeContent := `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:8080
  name: foo-cluster
contexts:
- context:
    cluster: foo-cluster
    user: foo-user
    namespace: bar
  name: foo-context
current-context: foo-context
kind: Config
users:
- name: foo-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - arg-1
      - arg-2
      command: foo-command
`
	err := createTestFile(emptyKubeConfig)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(emptyKubeConfig); err != nil {
			t.Error(err)
		}
	}()

	err = createTestFile(validKubeConfig)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(validKubeConfig); err != nil {
			t.Error(err)
		}
	}()

	if err := os.WriteFile(validKubeConfig, []byte(fakeContent), 0666); err != nil {
		t.Error(err)
	}

	os.Setenv("CONTAINER_SANDBOX_MOUNT_POINT", "C:\\var\\lib\\kubelet\\pods\\12345678-1234-1234-1234-123456789012")
	defer os.Unsetenv("CONTAINER_SANDBOX_MOUNT_POINT")

	tests := []struct {
		desc                     string
		kubeconfig               string
		enableWindowsHostProcess bool
		expectError              bool
		envVariableHasConfig     bool
		envVariableConfigIsValid bool
	}{
		{
			desc:                     "[success] valid kube config passed",
			kubeconfig:               validKubeConfig,
			enableWindowsHostProcess: false,
			expectError:              false,
			envVariableHasConfig:     false,
			envVariableConfigIsValid: false,
		},
		{
			desc:                     "[failure] invalid kube config passed",
			kubeconfig:               emptyKubeConfig,
			enableWindowsHostProcess: false,
			expectError:              true,
			envVariableHasConfig:     false,
			envVariableConfigIsValid: false,
		},
		{
			desc:                     "[failure] empty Kubeconfig under container sandbox mount path",
			kubeconfig:               "",
			enableWindowsHostProcess: true,
			expectError:              true,
			envVariableHasConfig:     false,
			envVariableConfigIsValid: false,
		},
	}

	for _, test := range tests {
		_, err := getKubeConfig(test.kubeconfig, test.enableWindowsHostProcess)
		receiveError := (err != nil)
		if test.expectError != receiveError {
			t.Errorf("desc: %s,\n input: %q, GetCloudProvider err: %v, expectErr: %v", test.desc, test.kubeconfig, err, test.expectError)
		}
	}
}

func createTestFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func TestGetUserNamePasswordFromSecret(t *testing.T) {
	tests := []struct {
		desc             string
		secretName       string
		secretNamespace  string
		expectedUsername string
		expectedPassword string
		expectedDomain   string
		expectedError    error
	}{
		{
			desc:          "kubeclient is nil",
			secretName:    "secretName",
			expectedError: fmt.Errorf("could not username and password from secret(secretName): KubeClient is nil"),
		},
	}

	d := NewFakeDriver()
	for _, test := range tests {
		username, password, domain, err := d.GetUserNamePasswordFromSecret(context.Background(), test.secretName, test.secretNamespace)
		assert.Equal(t, test.expectedUsername, username, "test[%s]: unexpected username", test.desc)
		assert.Equal(t, test.expectedPassword, password, "test[%s]: unexpected password", test.desc)
		assert.Equal(t, test.expectedDomain, domain, "test[%s]: unexpected domain", test.desc)
		assert.Equal(t, test.expectedError, err, "test[%s]: unexpected error", test.desc)
	}
}
