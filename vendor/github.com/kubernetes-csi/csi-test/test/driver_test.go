/*
Copyright 2017 Luis Pabón luis@portworx.com

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
package test

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-test/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Example simple driver
// This example assumes that your driver will create the server and listen on
// some unix domain socket or port for tests.
type simpleDriver struct {
	listener net.Listener
	server   *grpc.Server
	wg       sync.WaitGroup
}

func (s *simpleDriver) GetPluginCapabilities(context.Context, *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	// TODO: Return some simple Plugin Capabilities
	return &csi.GetPluginCapabilitiesResponse{}, nil
}

func (s *simpleDriver) Probe(context.Context, *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{}, nil
}

func (s *simpleDriver) GetPluginInfo(
	context.Context, *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          "simpleDriver",
		VendorVersion: "0.1.1",
		Manifest: map[string]string{
			"hello": "world",
		},
	}, nil
}

func (s *simpleDriver) goServe() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.server.Serve(s.listener)
	}()
}

func (s *simpleDriver) Address() string {
	return s.listener.Addr().String()
}

func (s *simpleDriver) Start() error {
	// Listen on a port assigned by the net package
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.listener = l

	// Create a new grpc server
	s.server = grpc.NewServer()

	csi.RegisterIdentityServer(s.server, s)
	reflection.Register(s.server)

	// Start listening for requests
	s.goServe()
	return nil
}

func (s *simpleDriver) Stop() {
	s.server.Stop()
	s.wg.Wait()
}

//
// Tests
//
func TestSimpleDriver(t *testing.T) {

	// Setup simple driver
	s := &simpleDriver{}
	err := s.Start()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}
	defer s.Stop()

	// Setup a connection to the driver
	conn, err := utils.Connect(s.Address())
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}
	defer conn.Close()

	// Make a call
	c := csi.NewIdentityClient(conn)
	r, err := c.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	// Verify
	name := r.GetName()
	if name != "simpleDriver" {
		t.Errorf("Unknown name: %s\n", name)
	}
}
