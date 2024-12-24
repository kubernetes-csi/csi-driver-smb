/*
Copyright 2024 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"testing"
)

func TestMain(t *testing.T) {
	// Set the version flag to true
	os.Args = []string{"cmd", "-ver"}

	// Capture stdout
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Replace exit function with mock function
	var exitCode int
	exit = func(code int) {
		exitCode = code
	}

	// Call main function
	main()

	// Restore stdout
	w.Close()
	os.Stdout = old
	exit = func(code int) {
		os.Exit(code)
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, but got %d", exitCode)
	}
}

func TestTrapClosedConnErr(t *testing.T) {
	tests := []struct {
		err         error
		expectedErr error
	}{
		{
			err:         net.ErrClosed,
			expectedErr: nil,
		},
		{
			err:         nil,
			expectedErr: nil,
		},
		{
			err:         fmt.Errorf("some error"),
			expectedErr: fmt.Errorf("some error"),
		},
	}

	for _, test := range tests {
		err := trapClosedConnErr(test.err)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Expected error %v, but got %v", test.expectedErr, err)
		}
	}
}
