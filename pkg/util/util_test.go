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
	"fmt"
	"testing"
	"time"
)

func TestWaitUntilTimeout(t *testing.T) {
	tests := []struct {
		desc        string
		timeout     time.Duration
		execFunc    ExecFunc
		timeoutFunc TimeoutFunc
		expectedErr error
	}{
		{
			desc:    "execFunc returns error",
			timeout: 1 * time.Second,
			execFunc: func() error {
				return fmt.Errorf("execFunc error")
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: fmt.Errorf("execFunc error"),
		},
		{
			desc:    "execFunc timeout",
			timeout: 1 * time.Second,
			execFunc: func() error {
				time.Sleep(2 * time.Second)
				return nil
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: fmt.Errorf("timeout error"),
		},
		{
			desc:    "execFunc completed successfully",
			timeout: 1 * time.Second,
			execFunc: func() error {
				return nil
			},
			timeoutFunc: func() error {
				return fmt.Errorf("timeout error")
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		err := WaitUntilTimeout(test.timeout, test.execFunc, test.timeoutFunc)
		if err != nil && (err.Error() != test.expectedErr.Error()) {
			t.Errorf("unexpected error: %v, expected error: %v", err, test.expectedErr)
		}
	}
}
