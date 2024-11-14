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
	"time"
)

// ExecFunc returns a exec function's output and error
type ExecFunc func() (err error)

// TimeoutFunc returns output and error if an ExecFunc timeout
type TimeoutFunc func() (err error)

// WaitUntilTimeout waits for the exec function to complete or return timeout error
func WaitUntilTimeout(timeout time.Duration, execFunc ExecFunc, timeoutFunc TimeoutFunc) error {
	// Create a channel to receive the result of the exec function
	done := make(chan bool)
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
