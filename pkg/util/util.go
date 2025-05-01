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

package util

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const MaxPathLengthWindows = 260

// control the number of concurrent powershell commands running on Windows node
var powershellCmdSem = make(chan struct{}, 3)

// ExecFunc returns a exec function's output and error
type ExecFunc func() (err error)

// TimeoutFunc returns output and error if an ExecFunc timeout
type TimeoutFunc func() (err error)

// WaitUntilTimeout waits for the exec function to complete or return timeout error
func WaitUntilTimeout(timeout time.Duration, execFunc ExecFunc, timeoutFunc TimeoutFunc) error {
	// Create a channel to receive the result of the exec function
	done := make(chan bool, 1)
	var err error

	// Start the exec function in a goroutine
	go func() {
		err = execFunc()
		done <- true
	}()

	// Wait for the function to complete or time out
	select {
	case <-done:
		return err
	case <-time.After(timeout):
		return timeoutFunc()
	}
}

func RunPowershellCmd(command string, envs ...string) ([]byte, error) {
	// acquire a semaphore to limit the number of concurrent operations
	powershellCmdSem <- struct{}{}
	defer func() { <-powershellCmdSem }()

	cmd := exec.Command("powershell", "-Mta", "-NoProfile", "-Command", command)
	cmd.Env = append(os.Environ(), envs...)
	klog.V(6).Infof("Executing command: %q", cmd.String())
	return cmd.CombinedOutput()
}

// SetKeyValueInMap set key/value pair in map
// key in the map is case insensitive, if key already exists, overwrite existing value
func SetKeyValueInMap(m map[string]string, key, value string) {
	if m == nil {
		return
	}
	for k := range m {
		if strings.EqualFold(k, key) {
			m[k] = value
			return
		}
	}
	m[key] = value
}
