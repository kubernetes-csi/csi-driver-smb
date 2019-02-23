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

//go:generate mockgen -package=driver -destination=driver.mock.go github.com/container-storage-interface/spec/lib/go/csi IdentityServer,ControllerServer,NodeServer

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	// ErrNoCredentials is the error when a secret is enabled but not passed in the request.
	ErrNoCredentials = errors.New("secret must be provided")
	// ErrAuthFailed is the error when the secret is incorrect.
	ErrAuthFailed = errors.New("authentication failed")
)

type CSIDriverServers struct {
	Controller csi.ControllerServer
	Identity   csi.IdentityServer
	Node       csi.NodeServer
}

// This is the key name in all the CSI secret objects.
const secretField = "secretKey"

// CSICreds is a driver specific secret type. Drivers can have a key-val pair of
// secrets. This mock driver has a single string secret with secretField as the
// key.
type CSICreds struct {
	CreateVolumeSecret              string
	DeleteVolumeSecret              string
	ControllerPublishVolumeSecret   string
	ControllerUnpublishVolumeSecret string
	NodeStageVolumeSecret           string
	NodePublishVolumeSecret         string
	CreateSnapshotSecret            string
	DeleteSnapshotSecret            string
}

type CSIDriver struct {
	listener net.Listener
	server   *grpc.Server
	servers  *CSIDriverServers
	wg       sync.WaitGroup
	running  bool
	lock     sync.Mutex
	creds    *CSICreds
}

func NewCSIDriver(servers *CSIDriverServers) *CSIDriver {
	return &CSIDriver{
		servers: servers,
	}
}

func (c *CSIDriver) goServe(started chan<- bool) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		started <- true
		err := c.server.Serve(c.listener)
		if err != nil {
			panic(err.Error())
		}
	}()
}

func (c *CSIDriver) Address() string {
	return c.listener.Addr().String()
}
func (c *CSIDriver) Start(l net.Listener) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Set listener
	c.listener = l

	// Create a new grpc server
	c.server = grpc.NewServer(
		grpc.UnaryInterceptor(c.callInterceptor),
	)

	// Register Mock servers
	if c.servers.Controller != nil {
		csi.RegisterControllerServer(c.server, c.servers.Controller)
	}
	if c.servers.Identity != nil {
		csi.RegisterIdentityServer(c.server, c.servers.Identity)
	}
	if c.servers.Node != nil {
		csi.RegisterNodeServer(c.server, c.servers.Node)
	}
	reflection.Register(c.server)

	// Start listening for requests
	waitForServer := make(chan bool)
	c.goServe(waitForServer)
	<-waitForServer
	c.running = true
	return nil
}

func (c *CSIDriver) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.running {
		return
	}

	c.server.Stop()
	c.wg.Wait()
}

func (c *CSIDriver) Close() {
	c.server.Stop()
}

func (c *CSIDriver) IsRunning() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.running
}

// SetDefaultCreds sets the default secrets for CSI creds.
func (c *CSIDriver) SetDefaultCreds() {
	c.creds = &CSICreds{
		CreateVolumeSecret:              "secretval1",
		DeleteVolumeSecret:              "secretval2",
		ControllerPublishVolumeSecret:   "secretval3",
		ControllerUnpublishVolumeSecret: "secretval4",
		NodeStageVolumeSecret:           "secretval5",
		NodePublishVolumeSecret:         "secretval6",
		CreateSnapshotSecret:            "secretval7",
		DeleteSnapshotSecret:            "secretval8",
	}
}

func (c *CSIDriver) callInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	err := c.authInterceptor(req)
	if err != nil {
		logGRPC(info.FullMethod, req, nil, err)
		return nil, err
	}
	rsp, err := handler(ctx, req)
	logGRPC(info.FullMethod, req, rsp, err)
	return rsp, err
}

func (c *CSIDriver) authInterceptor(req interface{}) error {
	if c.creds != nil {
		authenticated, authErr := isAuthenticated(req, c.creds)
		if !authenticated {
			if authErr == ErrNoCredentials {
				return status.Error(codes.InvalidArgument, authErr.Error())
			}
			if authErr == ErrAuthFailed {
				return status.Error(codes.Unauthenticated, authErr.Error())
			}
		}
	}
	return nil
}

func logGRPC(method string, request, reply interface{}, err error) {
	// Log JSON with the request and response for easier parsing
	logMessage := struct {
		Method   string
		Request  interface{}
		Response interface{}
		Error    string
	}{
		Method:   method,
		Request:  request,
		Response: reply,
	}
	if err != nil {
		logMessage.Error = err.Error()
	}
	msg, _ := json.Marshal(logMessage)
	fmt.Printf("gRPCCall: %s\n", msg)
}

func isAuthenticated(req interface{}, creds *CSICreds) (bool, error) {
	switch r := req.(type) {
	case *csi.CreateVolumeRequest:
		return authenticateCreateVolume(r, creds)
	case *csi.DeleteVolumeRequest:
		return authenticateDeleteVolume(r, creds)
	case *csi.ControllerPublishVolumeRequest:
		return authenticateControllerPublishVolume(r, creds)
	case *csi.ControllerUnpublishVolumeRequest:
		return authenticateControllerUnpublishVolume(r, creds)
	case *csi.NodeStageVolumeRequest:
		return authenticateNodeStageVolume(r, creds)
	case *csi.NodePublishVolumeRequest:
		return authenticateNodePublishVolume(r, creds)
	case *csi.CreateSnapshotRequest:
		return authenticateCreateSnapshot(r, creds)
	case *csi.DeleteSnapshotRequest:
		return authenticateDeleteSnapshot(r, creds)
	default:
		return true, nil
	}
}

func authenticateCreateVolume(req *csi.CreateVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.CreateVolumeSecret)
}

func authenticateDeleteVolume(req *csi.DeleteVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.DeleteVolumeSecret)
}

func authenticateControllerPublishVolume(req *csi.ControllerPublishVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.ControllerPublishVolumeSecret)
}

func authenticateControllerUnpublishVolume(req *csi.ControllerUnpublishVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.ControllerUnpublishVolumeSecret)
}

func authenticateNodeStageVolume(req *csi.NodeStageVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.NodeStageVolumeSecret)
}

func authenticateNodePublishVolume(req *csi.NodePublishVolumeRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.NodePublishVolumeSecret)
}

func authenticateCreateSnapshot(req *csi.CreateSnapshotRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.CreateSnapshotSecret)
}

func authenticateDeleteSnapshot(req *csi.DeleteSnapshotRequest, creds *CSICreds) (bool, error) {
	return credsCheck(req.GetSecrets(), creds.DeleteSnapshotSecret)
}

func credsCheck(secrets map[string]string, secretVal string) (bool, error) {
	if len(secrets) == 0 {
		return false, ErrNoCredentials
	}

	if secrets[secretField] != secretVal {
		return false, ErrAuthFailed
	}
	return true, nil
}
