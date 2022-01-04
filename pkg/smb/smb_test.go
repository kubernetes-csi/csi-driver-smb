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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	fakeNodeID = "fakeNodeID"
)

func NewFakeDriver() *Driver {
	return NewDriver(fakeNodeID, DefaultDriverName, true)
}

func TestNewFakeDriver(t *testing.T) {
	// Test New fake driver.
	d := NewDriver(fakeNodeID, DefaultDriverName, true)
	assert.NotNil(t, d)
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

func TestRun(t *testing.T) {
	testCases := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Successful run",
			testFunc: func(t *testing.T) {
				d := NewFakeDriver()
				d.Run("tcp://127.0.0.1:0", "", true)
			},
		},
		{
			name: "Successful run with node ID missing",
			testFunc: func(t *testing.T) {
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
