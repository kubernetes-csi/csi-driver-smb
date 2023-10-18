//go:build windows
// +build windows

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
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var basePath = "c:\\csi\\smbmounts"
var mutexes sync.Map

func lock(key string) func() {
	value, _ := mutexes.LoadOrStore(key, &sync.Mutex{})
	mtx := value.(*sync.Mutex)
	mtx.Lock()

	return func() { mtx.Unlock() }
}

// getRootMappingPath - returns root of smb share path or empty string if the path is invalid. For example:
//
//	\\hostname\share\subpath   =>   \\hostname\share, error is nil
//	\\hostname\share           =>   \\hostname\share, error is nil
//	\\hostname                 =>   '', error is 'remote path (\\hostname) is invalid'
func getRootMappingPath(path string) (string, error) {
	items := strings.Split(path, "\\")
	parts := []string{}
	for _, s := range items {
		if len(s) > 0 {
			parts = append(parts, s)
			if len(parts) == 2 {
				break
			}
		}
	}
	if len(parts) != 2 {
		return "", fmt.Errorf("remote path (%s) is invalid", path)
	}
	// parts[0] is a smb host name
	// parts[1] is a smb share name
	return strings.ToLower("\\\\" + parts[0] + "\\" + parts[1]), nil
}

// incementVolumeIDReferencesCount - adds new reference between mappingPath and remotePath if it doesn't exist.
// How it works:
//  1. MappingPath contains two components: hostname, sharename
//  2. We create directory in basePath related to each mappingPath. It will be used as container for references.
//     Example: c:\\csi\\smbmounts\\hostname\\sharename
//  3. Each reference is a file with name based on MD5 of volumeID. For debug it also will contains remotePath in body of the file.
//     So, in incementVolumeIDReferencesCount we create the file. In decrementRemotePathReferencesCount we remove the file.
//     Example: c:\\csi\\smbmounts\\hostname\\sharename\\092f1413e6c1d03af8b5da6f44619af8
func incementVolumeIDReferencesCount(mappingPath, remotePath string, volumeID string) error {
	remotePath = strings.TrimSuffix(remotePath, "\\")
	path := filepath.Join(basePath, strings.TrimPrefix(mappingPath, "\\\\"))
	if err := os.MkdirAll(path, os.ModeDir); err != nil {
		return err
	}
	filePath := filepath.Join(path, getMd5(volumeID))
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
	}()

	_, err = file.WriteString(remotePath)
	return err
}

// decrementVolumeIDReferencesCount - removes reference between mappingPath and remotePath.
// See incementVolumeIDReferencesCount to understand how references work.
func decrementVolumeIDReferencesCount(mappingPath, volumeID string) error {
	path := filepath.Join(basePath, strings.TrimPrefix(mappingPath, "\\\\"))
	if err := os.MkdirAll(path, os.ModeDir); err != nil {
		return err
	}
	filePath := filepath.Join(path, getMd5(volumeID))
	return os.Remove(filePath)
}

// getVolumeIDReferencesCount - returns count of references between mappingPath and remotePath.
// See incementVolumeIDReferencesCount to understand how references work.
func getVolumeIDReferencesCount(mappingPath string) int {
	path := filepath.Join(basePath, strings.TrimPrefix(mappingPath, "\\\\"))
	if os.MkdirAll(path, os.ModeDir) != nil {
		return -1
	}
	files, err := os.ReadDir(path)
	if err != nil {
		return -1
	}
	return len(files)
}

func getMd5(path string) string {
	data := []byte(strings.ToLower(path))
	return fmt.Sprintf("%x", md5.Sum(data))
}
