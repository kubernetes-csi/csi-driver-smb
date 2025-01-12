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
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	mount "k8s.io/mount-utils"

	csicommon "github.com/kubernetes-csi/csi-driver-smb/pkg/csi-common"
	"github.com/kubernetes-csi/csi-driver-smb/pkg/mounter"

	azcache "sigs.k8s.io/cloud-provider-azure/pkg/cache"
)

const (
	DefaultDriverName         = "smb.csi.k8s.io"
	usernameField             = "username"
	passwordField             = "password"
	sourceField               = "source"
	subDirField               = "subdir"
	domainField               = "domain"
	mountOptionsField         = "mountoptions"
	secretNameField           = "secretname"
	secretNamespaceField      = "secretnamespace"
	paramOnDelete             = "ondelete"
	defaultDomainName         = "AZURE"
	ephemeralField            = "csi.storage.k8s.io/ephemeral"
	podNamespaceField         = "csi.storage.k8s.io/pod.namespace"
	pvcNameKey                = "csi.storage.k8s.io/pvc/name"
	pvcNamespaceKey           = "csi.storage.k8s.io/pvc/namespace"
	pvNameKey                 = "csi.storage.k8s.io/pv/name"
	pvcNameMetadata           = "${pvc.metadata.name}"
	pvcNamespaceMetadata      = "${pvc.metadata.namespace}"
	pvNameMetadata            = "${pv.metadata.name}"
	DefaultKrb5CCName         = "krb5cc_"
	DefaultKrb5CacheDirectory = "/var/lib/kubelet/kerberos/"
	retain                    = "retain"
	archive                   = "archive"
	fileMode                  = "file_mode"
	dirMode                   = "dir_mode"
	defaultFileMode           = "0777"
	defaultDirMode            = "0777"
	trueValue                 = "true"
)

var supportedOnDeleteValues = []string{"", "delete", retain, archive}

// DriverOptions defines driver parameters specified in driver deployment
type DriverOptions struct {
	NodeID               string
	DriverName           string
	EnableGetVolumeStats bool
	// this only applies to Windows node
	RemoveSMBMappingDuringUnmount bool
	WorkingMountDir               string
	VolStatsCacheExpireInMinutes  int
	Krb5CacheDirectory            string
	Krb5Prefix                    string
	DefaultOnDeletePolicy         string
	RemoveArchivedVolumePath      bool
	EnableWindowsHostProcess      bool
	Kubeconfig                    string
}

// Driver implements all interfaces of CSI drivers
type Driver struct {
	csicommon.CSIDriver
	// Embed UnimplementedXXXServer to ensure the driver returns Unimplemented for any
	// new RPC methods that might be introduced in future versions of the spec.
	csi.UnimplementedControllerServer
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer

	mounter *mount.SafeFormatAndMount
	// A map storing all volumes with ongoing operations so that additional operations
	// for that same volume (as defined by VolumeID) return an Aborted error
	volumeLocks          *volumeLocks
	workingMountDir      string
	enableGetVolumeStats bool
	// a timed cache storing volume stats <volumeID, volumeStats>
	volStatsCache azcache.Resource
	// a timed cache storing volume deletion records <volumeID, "">
	volDeletionCache azcache.Resource
	// this only applies to Windows node
	removeSMBMappingDuringUnmount bool
	krb5CacheDirectory            string
	krb5Prefix                    string
	defaultOnDeletePolicy         string
	removeArchivedVolumePath      bool
	enableWindowsHostProcess      bool
	kubeconfig                    string
	kubeClient                    kubernetes.Interface
}

// NewDriver Creates a NewCSIDriver object. Assumes vendor version is equal to driver version &
// does not support optional driver plugin info manifest field. Refer to CSI spec for more details.
func NewDriver(options *DriverOptions) *Driver {
	driver := Driver{}
	driver.Name = options.DriverName
	driver.Version = driverVersion
	driver.NodeID = options.NodeID
	driver.enableGetVolumeStats = options.EnableGetVolumeStats
	driver.removeSMBMappingDuringUnmount = options.RemoveSMBMappingDuringUnmount
	driver.removeArchivedVolumePath = options.RemoveArchivedVolumePath
	driver.workingMountDir = options.WorkingMountDir
	driver.enableWindowsHostProcess = options.EnableWindowsHostProcess
	driver.kubeconfig = options.Kubeconfig
	driver.volumeLocks = newVolumeLocks()

	driver.krb5CacheDirectory = options.Krb5CacheDirectory
	if driver.krb5CacheDirectory == "" {
		driver.krb5CacheDirectory = DefaultKrb5CacheDirectory
	}
	driver.krb5Prefix = options.Krb5Prefix
	if driver.krb5Prefix == "" {
		driver.krb5Prefix = DefaultKrb5CCName
	}

	if options.VolStatsCacheExpireInMinutes <= 0 {
		options.VolStatsCacheExpireInMinutes = 10 // default expire in 10 minutes
	}
	var err error
	getter := func(_ string) (interface{}, error) { return nil, nil }
	if driver.volStatsCache, err = azcache.NewTimedCache(time.Duration(options.VolStatsCacheExpireInMinutes)*time.Minute, getter, false); err != nil {
		klog.Fatalf("%v", err)
	}
	if driver.volDeletionCache, err = azcache.NewTimedCache(time.Minute, getter, false); err != nil {
		klog.Fatalf("%v", err)
	}

	kubeCfg, err := getKubeConfig(driver.kubeconfig, driver.enableWindowsHostProcess)
	if err == nil && kubeCfg != nil {
		if driver.kubeClient, err = kubernetes.NewForConfig(kubeCfg); err != nil {
			klog.Warningf("NewForConfig failed with error: %v", err)
		}
	} else {
		klog.Warningf("get kubeconfig(%s) failed with error: %v", driver.kubeconfig, err)
	}
	return &driver
}

// Run driver initialization
func (d *Driver) Run(endpoint, _ string, testMode bool) {
	versionMeta, err := GetVersionYAML(d.Name)
	if err != nil {
		klog.Fatalf("%v", err)
	}
	klog.V(2).Infof("\nDRIVER INFORMATION:\n-------------------\n%s\n\nStreaming logs below:", versionMeta)

	d.mounter, err = mounter.NewSafeMounter(d.enableWindowsHostProcess, d.removeSMBMappingDuringUnmount)
	if err != nil {
		klog.Fatalf("Failed to get safe mounter. Error: %v", err)
	}

	// Initialize default library driver
	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
			csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
			csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		})

	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_MULTI_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})

	nodeCap := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_SINGLE_NODE_MULTI_WRITER,
		csi.NodeServiceCapability_RPC_VOLUME_MOUNT_GROUP,
	}
	if d.enableGetVolumeStats {
		nodeCap = append(nodeCap, csi.NodeServiceCapability_RPC_GET_VOLUME_STATS)
	}
	d.AddNodeServiceCapabilities(nodeCap)

	s := csicommon.NewNonBlockingGRPCServer()
	// Driver d act as IdentityServer, ControllerServer and NodeServer
	s.Start(endpoint, d, d, d, testMode)
	s.Wait()
}

// GetUserNamePasswordFromSecret get storage account key from k8s secret
// return <username, password, domain, error>
func (d *Driver) GetUserNamePasswordFromSecret(ctx context.Context, secretName, secretNamespace string) (string, string, string, error) {
	if d.kubeClient == nil {
		return "", "", "", fmt.Errorf("could not username and password from secret(%s): KubeClient is nil", secretName)
	}

	secret, err := d.kubeClient.CoreV1().Secrets(secretNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", "", fmt.Errorf("could not get secret(%v): %v", secretName, err)
	}

	username := strings.TrimSpace(string(secret.Data[usernameField][:]))
	password := strings.TrimSpace(string(secret.Data[passwordField][:]))
	domain := strings.TrimSpace(string(secret.Data[domainField][:]))
	return username, password, domain, nil
}

func IsCorruptedDir(dir string) bool {
	_, pathErr := mount.PathExists(dir)
	return pathErr != nil && mount.IsCorruptedMnt(pathErr)
}

// getMountOptions get mountOptions value from a map
func getMountOptions(context map[string]string) string {
	for k, v := range context {
		switch strings.ToLower(k) {
		case mountOptionsField:
			return v
		}
	}
	return ""
}

func hasGuestMountOptions(options []string) bool {
	for _, v := range options {
		if v == "guest" {
			return true
		}
	}
	return false
}

// setKeyValueInMap set key/value pair in map
// key in the map is case insensitive, if key already exists, overwrite existing value
func setKeyValueInMap(m map[string]string, key, value string) {
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

// replaceWithMap replace key with value for str
func replaceWithMap(str string, m map[string]string) string {
	for k, v := range m {
		if k != "" {
			str = strings.ReplaceAll(str, k, v)
		}
	}
	return str
}

func validateOnDeleteValue(onDelete string) error {
	for _, v := range supportedOnDeleteValues {
		if strings.EqualFold(v, onDelete) {
			return nil
		}
	}

	return fmt.Errorf("invalid value %s for OnDelete, supported values are %v", onDelete, supportedOnDeleteValues)
}

// appendMountOptions appends extra mount options to the given mount options
func appendMountOptions(mountOptions []string, extraMountOptions map[string]string) []string {
	// stores the mount options already included in mountOptions
	included := make(map[string]bool)
	for _, mountOption := range mountOptions {
		for k := range extraMountOptions {
			if strings.HasPrefix(mountOption, k) {
				included[k] = true
			}
		}
	}

	allMountOptions := mountOptions
	for k, v := range extraMountOptions {
		if _, isIncluded := included[k]; !isIncluded {
			if v != "" {
				allMountOptions = append(allMountOptions, fmt.Sprintf("%s=%s", k, v))
			} else {
				allMountOptions = append(allMountOptions, k)
			}
		}
	}
	return allMountOptions
}

// getRootDir returns the root directory of the given directory
func getRootDir(path string) string {
	parts := strings.Split(path, "/")
	return parts[0]
}

func getKubeConfig(kubeconfig string, enableWindowsHostProcess bool) (config *rest.Config, err error) {
	if kubeconfig != "" {
		if config, err = clientcmd.BuildConfigFromFlags("", kubeconfig); err != nil {
			return nil, err
		}
	} else {
		if config, err = inClusterConfig(enableWindowsHostProcess); err != nil {
			return nil, err
		}
	}
	return config, err
}

// inClusterConfig is copied from https://github.com/kubernetes/client-go/blob/b46677097d03b964eab2d67ffbb022403996f4d4/rest/config.go#L507-L541
// When using Windows HostProcess containers, the path "/var/run/secrets/kubernetes.io/serviceaccount/" is under host, not container.
// Then the token and ca.crt files would be not found.
// An environment variable $CONTAINER_SANDBOX_MOUNT_POINT is set upon container creation and provides the absolute host path to the container volume.
// See https://kubernetes.io/docs/tasks/configure-pod-container/create-hostprocess-pod/#volume-mounts for more details.
func inClusterConfig(enableWindowsHostProcess bool) (*rest.Config, error) {
	var (
		tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	)
	if enableWindowsHostProcess {
		containerSandboxMountPath := os.Getenv("CONTAINER_SANDBOX_MOUNT_POINT")
		if len(containerSandboxMountPath) == 0 {
			return nil, errors.New("unable to load in-cluster configuration, containerSandboxMountPath must be defined")
		}
		tokenFile = filepath.Join(containerSandboxMountPath, tokenFile)
		rootCAFile = filepath.Join(containerSandboxMountPath, rootCAFile)
	}

	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, rest.ErrNotInCluster
	}

	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	tlsClientConfig := rest.TLSClientConfig{}

	if _, err := certutil.NewPool(rootCAFile); err != nil {
		klog.Errorf("Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &rest.Config{
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(token),
		BearerTokenFile: tokenFile,
	}, nil
}
