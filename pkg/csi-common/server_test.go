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

package csicommon

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestNewNonBlockingGRPCServer(t *testing.T) {
	s := NewNonBlockingGRPCServer()
	assert.NotNil(t, s)
}

func TestStart(t *testing.T) {
	s := NewNonBlockingGRPCServer()
	// sleep a while to avoid race condition in unit test
	time.Sleep(time.Millisecond * 500)
	s.Start("tcp://127.0.0.1:0", nil, nil, nil, true)
	time.Sleep(time.Millisecond * 500)
}

func TestServe(t *testing.T) {
	s := nonBlockingGRPCServer{}
	s.server = grpc.NewServer()
	s.wg = sync.WaitGroup{}
	//need to add one here as the actual also requires one.
	s.wg.Add(1)
	s.serve("tcp://127.0.0.1:0", nil, nil, nil, true)
}

func TestWait(t *testing.T) {
	s := nonBlockingGRPCServer{}
	s.server = grpc.NewServer()
	s.wg = sync.WaitGroup{}
	s.Wait()
}

func TestStop(t *testing.T) {
	s := nonBlockingGRPCServer{}
	s.server = grpc.NewServer()
	s.Stop()
}

func TestForceStop(t *testing.T) {
	s := nonBlockingGRPCServer{}
	s.server = grpc.NewServer()
	s.ForceStop()
}
