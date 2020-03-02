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

package azurefile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendDefaultMountOptions(t *testing.T) {
	tests := []struct {
		options  []string
		expected []string
	}{
		{
			options: []string{"dir_mode=0777"},
			expected: []string{"dir_mode=0777",
				fmt.Sprintf("%s=%s", fileMode, defaultFileMode),
				fmt.Sprintf("%s=%s", vers, defaultVers)},
		},
		{
			options: []string{"file_mode=0777"},
			expected: []string{"file_mode=0777",
				fmt.Sprintf("%s=%s", dirMode, defaultDirMode),
				fmt.Sprintf("%s=%s", vers, defaultVers)},
		},
		{
			options: []string{"vers=2.1"},
			expected: []string{"vers=2.1",
				fmt.Sprintf("%s=%s", fileMode, defaultFileMode),
				fmt.Sprintf("%s=%s", dirMode, defaultDirMode)},
		},
		{
			options: []string{""},
			expected: []string{"", fmt.Sprintf("%s=%s",
				fileMode, defaultFileMode),
				fmt.Sprintf("%s=%s", dirMode, defaultDirMode),
				fmt.Sprintf("%s=%s", vers, defaultVers)},
		},
		{
			options:  []string{"file_mode=0777", "dir_mode=0777"},
			expected: []string{"file_mode=0777", "dir_mode=0777", fmt.Sprintf("%s=%s", vers, defaultVers)},
		},
	}

	for _, test := range tests {
		result := appendDefaultMountOptions(test.options)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("input: %q, appendDefaultMountOptions result: %q, expected: %q", test.options, result, test.expected)
		}
	}
}

func TestGetFileShareInfo(t *testing.T) {
	tests := []struct {
		options   string
		expected1 string
		expected2 string
		expected3 string
		expected4 error
	}{
		{
			options:   "rg#f5713de20cde511e8ba4900#pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41",
			expected1: "rg",
			expected2: "f5713de20cde511e8ba4900",
			expected3: "pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41",
			expected4: nil,
		},
		{
			options:   "rg#f5713de20cde511e8ba4900",
			expected1: "",
			expected2: "",
			expected3: "",
			expected4: fmt.Errorf("error parsing volume id: \"rg#f5713de20cde511e8ba4900\", should at least contain two #"),
		},
		{
			options:   "rg",
			expected1: "",
			expected2: "",
			expected3: "",
			expected4: fmt.Errorf("error parsing volume id: \"rg\", should at least contain two #"),
		},
		{
			options:   "",
			expected1: "",
			expected2: "",
			expected3: "",
			expected4: fmt.Errorf("error parsing volume id: \"\", should at least contain two #"),
		},
	}

	for _, test := range tests {
		result1, result2, result3, result4 := getFileShareInfo(test.options)
		if !reflect.DeepEqual(result1, test.expected1) || !reflect.DeepEqual(result2, test.expected2) ||
			!reflect.DeepEqual(result3, test.expected3) || !reflect.DeepEqual(result4, test.expected4) {
			t.Errorf("input: %q, getFileShareInfo result1: %q, expected1: %q, result2: %q, expected2: %q, result3: %q, expected3: %q, result4: %q, expected4: %q", test.options, result1, test.expected1, result2, test.expected2,
				result3, test.expected3, result4, test.expected4)
		}
	}
}

func TestGetStorageAccount(t *testing.T) {
	emptyAccountKeyMap := map[string]string{
		"accountname": "testaccount",
		"accountkey":  "",
	}

	emptyAccountNameMap := map[string]string{
		"azurestorageaccountname": "",
		"azurestorageaccountkey":  "testkey",
	}

	emptyAzureAccountKeyMap := map[string]string{
		"azurestorageaccountname": "testaccount",
		"azurestorageaccountkey":  "",
	}

	emptyAzureAccountNameMap := map[string]string{
		"azurestorageaccountname": "",
		"azurestorageaccountkey":  "testkey",
	}

	tests := []struct {
		options   map[string]string
		expected1 string
		expected2 string
		expected3 error
	}{
		{
			options: map[string]string{
				"accountname": "testaccount",
				"accountkey":  "testkey",
			},
			expected1: "testaccount",
			expected2: "testkey",
			expected3: nil,
		},
		{
			options: map[string]string{
				"azurestorageaccountname": "testaccount",
				"azurestorageaccountkey":  "testkey",
			},
			expected1: "testaccount",
			expected2: "testkey",
			expected3: nil,
		},
		{
			options: map[string]string{
				"accountname": "",
				"accountkey":  "",
			},
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("could not find accountname or azurestorageaccountname field secrets(map[accountname: accountkey:])"),
		},
		{
			options:   emptyAccountKeyMap,
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("could not find accountkey or azurestorageaccountkey field in secrets(%v)", emptyAccountKeyMap),
		},
		{
			options:   emptyAccountNameMap,
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("could not find accountname or azurestorageaccountname field secrets(%v)", emptyAccountNameMap),
		},
		{
			options:   emptyAzureAccountKeyMap,
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("could not find accountkey or azurestorageaccountkey field in secrets(%v)", emptyAzureAccountKeyMap),
		},
		{
			options:   emptyAzureAccountNameMap,
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("could not find accountname or azurestorageaccountname field secrets(%v)", emptyAzureAccountNameMap),
		},
		{
			options:   nil,
			expected1: "",
			expected2: "",
			expected3: fmt.Errorf("unexpected: getStorageAccount secrets is nil"),
		},
	}

	for _, test := range tests {
		result1, result2, result3 := getStorageAccount(test.options)
		if !reflect.DeepEqual(result1, test.expected1) || !reflect.DeepEqual(result2, test.expected2) {
			t.Errorf("input: %q, getStorageAccount result1: %q, expected1: %q, result2: %q, expected2: %q, result3: %q, expected3: %q", test.options, result1, test.expected1, result2, test.expected2,
				result3, test.expected3)
		} else {
			if result1 == "" || result2 == "" {
				assert.Error(t, result3)
			}
		}
	}
}

func TestGetValidFileShareName(t *testing.T) {
	tests := []struct {
		volumeName string
		expected   string
	}{
		{
			volumeName: "aqz",
			expected:   "aqz",
		},
		{
			volumeName: "029",
			expected:   "029",
		},
		{
			volumeName: "a--z",
			expected:   "a-z",
		},
		{
			volumeName: "A2Z",
			expected:   "a2z",
		},
		{
			volumeName: "1234567891234567891234567891234567891234567891234567891234567891",
			expected:   "123456789123456789123456789123456789123456789123456789123456789",
		},
	}

	for _, test := range tests {
		result := getValidFileShareName(test.volumeName)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("input: %q, getValidFileShareName result: %q, expected: %q", test.volumeName, result, test.expected)
		}
	}
}

func TestCheckShareNameBeginAndEnd(t *testing.T) {
	tests := []struct {
		fileShareName string
		expected      bool
	}{
		{
			fileShareName: "aqz",
			expected:      true,
		},
		{
			fileShareName: "029",
			expected:      true,
		},
		{
			fileShareName: "a-9",
			expected:      true,
		},
		{
			fileShareName: "0-z",
			expected:      true,
		},
		{
			fileShareName: "-1-",
			expected:      false,
		},
		{
			fileShareName: ":1p",
			expected:      false,
		},
	}

	for _, test := range tests {
		result := checkShareNameBeginAndEnd(test.fileShareName)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("input: %q, checkShareNameBeginAndEnd result: %v, expected: %v", test.fileShareName, result, test.expected)
		}
	}
}

func TestGetSnapshot(t *testing.T) {
	tests := []struct {
		options   string
		expected1 string
		expected2 error
	}{
		{
			options:   "rg#f123#csivolumename#2019-08-22T07:17:53.0000000Z",
			expected1: "2019-08-22T07:17:53.0000000Z",
			expected2: nil,
		},
		{
			options:   "rg#f123#csivolumename",
			expected1: "",
			expected2: fmt.Errorf("error parsing volume id: \"rg#f123#csivolumename\", should at least contain three #"),
		},
		{
			options:   "rg#f123",
			expected1: "",
			expected2: fmt.Errorf("error parsing volume id: \"rg#f123\", should at least contain three #"),
		},
		{
			options:   "rg",
			expected1: "",
			expected2: fmt.Errorf("error parsing volume id: \"rg\", should at least contain three #"),
		},
		{
			options:   "",
			expected1: "",
			expected2: fmt.Errorf("error parsing volume id: \"\", should at least contain three #"),
		},
	}

	for _, test := range tests {
		result1, result2 := getSnapshot(test.options)
		if !reflect.DeepEqual(result1, test.expected1) || !reflect.DeepEqual(result2, test.expected2) {
			t.Errorf("input: %q, getSnapshot result1: %q, expected1: %q, result2: %q, expected2: %q, ", test.options, result1, test.expected1, result2, test.expected2)
		}
	}
}

func TestIsCorruptedDir(t *testing.T) {
	existingMountPath, err := ioutil.TempDir(os.TempDir(), "csi-mount-test")
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
