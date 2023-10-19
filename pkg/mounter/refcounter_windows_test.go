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

package mounter

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLockUnlock(t *testing.T) {
	key := "resource name"

	unlock := lock(key)
	defer unlock()

	_, loaded := mutexes.Load(key)
	assert.True(t, loaded)
}

func TestLockLockedResource(t *testing.T) {
	locked := true
	unlock := lock("a")
	go func() {
		time.Sleep(500 * time.Microsecond)
		locked = false
		unlock()
	}()

	// try to lock already locked resource
	unlock2 := lock("a")
	defer unlock2()
	if locked {
		assert.Fail(t, "access to locked resource")
	}
}

func TestLockDifferentKeys(t *testing.T) {
	unlocka := lock("a")
	unlockb := lock("b")
	unlocka()
	unlockb()
}

func TestGetRootMappingPath(t *testing.T) {
	testCases := []struct {
		remote       string
		expectResult string
		expectError  bool
	}{
		{
			remote:       "",
			expectResult: "",
			expectError:  true,
		},
		{
			remote:       "hostname",
			expectResult: "",
			expectError:  true,
		},
		{
			remote:       "\\\\hostname\\path",
			expectResult: "\\\\hostname\\path",
			expectError:  false,
		},
		{
			remote:       "\\\\hostname\\path\\",
			expectResult: "\\\\hostname\\path",
			expectError:  false,
		},
		{
			remote:       "\\\\hostname\\path\\subpath",
			expectResult: "\\\\hostname\\path",
			expectError:  false,
		},
	}
	for _, tc := range testCases {
		result, err := getRootMappingPath(tc.remote)
		if tc.expectError && err == nil {
			t.Errorf("Expected error but getRootMappingPath returned a nil error")
		}
		if !tc.expectError {
			if err != nil {
				t.Errorf("Expected no errors but getRootMappingPath returned error: %v", err)
			}
			if tc.expectResult != result {
				t.Errorf("Expected (%s) but getRootMappingPath returned (%s)", tc.expectResult, result)
			}
		}
	}
}

func TestVolumeIDReferencesCounter(t *testing.T) {
	testCases := []struct {
		path1 string
		path2 string
	}{
		{
			path1: "\\\\servername\\share\\subpath\\1",
			path2: "\\\\servername\\share\\subpath\\2",
		},
		{
			path1: "\\\\servername\\share",
			path2: "\\\\servername\\share",
		},
	}
	for _, tc := range testCases {
		remotePath1 := tc.path1
		remotePath2 := tc.path2
		mappingPath, err := getRootMappingPath(remotePath1)
		assert.Nil(t, err)

		basePath = os.Getenv("TEMP") + "\\TestMappingPathCounter"
		os.RemoveAll(basePath)
		defer func() {
			// cleanup temp folder
			os.RemoveAll(basePath)
		}()

		// by default we have no any files in `mappingPath`. So, `count` should be zero
		assert.Zero(t, getVolumeIDReferencesCount(mappingPath))
		// add reference to `remotePath1`. So, `count` should be equal `1`
		assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath1, "vol1"))
		assert.Equal(t, 1, getVolumeIDReferencesCount(mappingPath))
		// add reference to `remotePath2`. So, `count` should be equal `2`
		assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath2, "vol2"))
		assert.Equal(t, 2, getVolumeIDReferencesCount(mappingPath))
		// remove reference to `remotePath1`. So, `count` should be equal `1`
		assert.Nil(t, decrementVolumeIDReferencesCount(mappingPath, "vol1"))
		assert.Equal(t, 1, getVolumeIDReferencesCount(mappingPath))
		// remove reference to `remotePath2`. So, `count` should be equal `0`
		assert.Nil(t, decrementVolumeIDReferencesCount(mappingPath, "vol2"))
		assert.Zero(t, getVolumeIDReferencesCount(mappingPath))
	}
}

func TestIncementVolumeIDReferencesCount(t *testing.T) {
	remotePath := "\\\\servername\\share\\subpath"
	mappingPath, err := getRootMappingPath(remotePath)
	assert.Nil(t, err)

	basePath = os.Getenv("TEMP") + "\\TestMappingPathCounter"
	os.RemoveAll(basePath)
	defer func() {
		// cleanup temp folder
		os.RemoveAll(basePath)
	}()

	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol1"))

	mappingPathContainer := basePath + "\\servername\\share"
	if dir, err := os.Stat(mappingPathContainer); os.IsNotExist(err) || !dir.IsDir() {
		t.Error("mapping file container does not exist")
	}

	reference := mappingPathContainer + "\\" + getMd5("vol1")
	if file, err := os.Stat(reference); os.IsNotExist(err) || file.IsDir() {
		t.Error("reference file does not exist")
	}
}

func TestDecrementVolumeIDReferencesCount(t *testing.T) {
	remotePath := "\\\\servername\\share\\subpath"
	mappingPath, err := getRootMappingPath(remotePath)
	assert.Nil(t, err)

	basePath = os.Getenv("TEMP") + "\\TestMappingPathCounter"
	os.RemoveAll(basePath)
	defer func() {
		// cleanup temp folder
		os.RemoveAll(basePath)
	}()

	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol1"))
	assert.Nil(t, decrementVolumeIDReferencesCount(mappingPath, "vol1"))

	mappingPathContainer := basePath + "\\servername\\share"
	if dir, err := os.Stat(mappingPathContainer); os.IsNotExist(err) || !dir.IsDir() {
		t.Error("mapping file container does not exist")
	}

	reference := mappingPathContainer + "\\" + getMd5("vol1")
	if _, err := os.Stat(reference); os.IsExist(err) {
		t.Error("reference file exists")
	}
}

func TestMultiplyCallsOfIncementVolumeIDReferencesCount(t *testing.T) {
	remotePath := "\\\\servername\\share\\subpath"
	mappingPath, err := getRootMappingPath(remotePath)
	assert.Nil(t, err)

	basePath = os.Getenv("TEMP") + "\\TestMappingPathCounter"
	os.RemoveAll(basePath)
	defer func() {
		// cleanup temp folder
		os.RemoveAll(basePath)
	}()

	assert.Zero(t, getVolumeIDReferencesCount(mappingPath))
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	// next calls of `incementMappingPathCount` with the same arguments should be ignored
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	assert.Equal(t, 1, getVolumeIDReferencesCount(mappingPath))
}

func TestMultiplyCallsOfDecrementVolumeIDReferencesCount(t *testing.T) {
	remotePath := "\\\\servername\\share\\subpath"
	mappingPath, err := getRootMappingPath(remotePath)
	assert.Nil(t, err)

	basePath = os.Getenv("TEMP") + "\\TestMappingPathCounter"
	os.RemoveAll(basePath)
	defer func() {
		// cleanup temp folder
		os.RemoveAll(basePath)
	}()

	assert.Zero(t, getVolumeIDReferencesCount(mappingPath))
	assert.Nil(t, incementVolumeIDReferencesCount(mappingPath, remotePath, "vol"))
	assert.Nil(t, decrementVolumeIDReferencesCount(mappingPath, "vol"))
	assert.NotNil(t, decrementVolumeIDReferencesCount(mappingPath, "vol"))
}
