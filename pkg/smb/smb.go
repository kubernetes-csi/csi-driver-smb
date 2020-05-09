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
	"bytes"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	azs "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/azure-storage-file-go/azfile"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pborman/uuid"
	"github.com/rubiojr/go-vhd/vhd"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/volume/util"
	"k8s.io/legacy-cloud-providers/azure"
	"k8s.io/utils/mount"

	csicommon "github.com/csi-driver/csi-driver-smb/pkg/csi-common"
	"github.com/csi-driver/csi-driver-smb/pkg/mounter"
)

const (
	DriverName         = "file.csi.azure.com"
	separator          = "#"
	volumeIDTemplate   = "%s#%s#%s#%s"
	secretNameTemplate = "azure-storage-account-%s-secret"
	serviceURLTemplate = "https://%s.file.%s"
	fileURLTemplate    = "https://%s.file.%s/%s/%s"
	fileMode           = "file_mode"
	dirMode            = "dir_mode"
	vers               = "vers"
	defaultFileMode    = "0777"
	defaultDirMode     = "0777"
	defaultVers        = "3.0"

	// See https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-shares--directories--files--and-metadata#share-names
	fileShareNameMinLength = 3
	fileShareNameMaxLength = 63

	minimumPremiumShareSize = 100 // GB
	// Minimum size of Azure Premium Files is 100GiB
	// See https://docs.microsoft.com/en-us/azure/storage/files/storage-files-planning#provisioned-shares
	defaultAzureFileQuota = 100

	// key of snapshot name in metadata
	snapshotNameKey = "initiator"

	shareNameField           = "sharename"
	diskNameField            = "diskname"
	fsTypeField              = "fstype"
	secretNamespaceField     = "secretnamespace"
	storeAccountKeyField     = "storeaccountkey"
	defaultSecretAccountName = "azurestorageaccountname"
	defaultSecretAccountKey  = "azurestorageaccountkey"
	defaultSecretNamespace   = "default"
	proxyMount               = "proxy-mount"
	cifs                     = "cifs"
	metaDataNode             = "node"

	accountNotProvisioned = "StorageAccountIsNotProvisioned"
	tooManyRequests       = "TooManyRequests"
	shareNotFound         = "The specified share does not exist"
)

// Driver implements all interfaces of CSI drivers
type Driver struct {
	csicommon.CSIDriver
	cloud   *azure.Cloud
	mounter *mount.SafeFormatAndMount
	// lock per volume attach (only for vhd disk feature)
	volLockMap *lockMap
}

// NewDriver Creates a NewCSIDriver object. Assumes vendor version is equal to driver version &
// does not support optional driver plugin info manifest field. Refer to CSI spec for more details.
func NewDriver(nodeID string) *Driver {
	driver := Driver{}
	driver.Name = DriverName
	driver.Version = driverVersion
	driver.NodeID = nodeID
	driver.volLockMap = newLockMap()
	return &driver
}

// Run driver initialization
func (d *Driver) Run(endpoint, kubeconfig string) {
	versionMeta, err := GetVersionYAML()
	if err != nil {
		klog.Fatalf("%v", err)
	}
	klog.Infof("\nDRIVER INFORMATION:\n-------------------\n%s\n\nStreaming logs below:", versionMeta)

	cloud, err := GetCloudProvider(kubeconfig)
	if err != nil || cloud.TenantID == "" || cloud.SubscriptionID == "" {
		klog.Fatalf("failed to get Azure Cloud Provider, error: %v", err)
	}
	d.cloud = cloud

	if d.NodeID == "" {
		// Disable UseInstanceMetadata for controller to mitigate a timeout issue using IMDS
		// https://github.com/kubernetes-sigs/azuredisk-csi-driver/issues/168
		klog.Infoln("disable UseInstanceMetadata for controller")
		d.cloud.Config.UseInstanceMetadata = false
	}

	d.mounter, err = mounter.NewSafeMounter()
	if err != nil {
		klog.Fatalf("Failed to get safe mounter. Error: %v", err)
	}

	// Initialize default library driver
	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
			//csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
			csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		})
	d.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})

	d.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
	})

	s := csicommon.NewNonBlockingGRPCServer()
	// Driver d act as IdentityServer, ControllerServer and NodeServer
	s.Start(endpoint, d, d, d)
	s.Wait()
}

func (d *Driver) checkFileShareCapacity(accountName, accountKey, fileShareName string, requestGiB int) error {
	fileClient, err := d.getFileSvcClient(accountName, accountKey)
	if err != nil {
		return err
	}
	resp, err := fileClient.ListShares(azs.ListSharesParameters{Prefix: fileShareName})
	if err != nil {
		return fmt.Errorf("error listing file shares: %v", err)
	}
	for _, share := range resp.Shares {
		if share.Name == fileShareName && share.Properties.Quota != requestGiB {
			return status.Errorf(codes.AlreadyExists, "the request volume already exists, but its capacity(%v) is different from (%v)", share.Properties.Quota, requestGiB)
		}
	}

	return nil
}

func (d *Driver) checkFileShareExists(accountName, resourceGroup, name string) (bool, error) {
	// find the access key with this account
	accountKey, err := d.cloud.GetStorageAccesskey(accountName, resourceGroup)
	if err != nil {
		return false, fmt.Errorf("error getting storage key for storage account %s: %v", accountName, err)
	}

	fileClient, err := d.getFileSvcClient(accountName, accountKey)
	if err != nil {
		return false, err
	}
	return fileClient.GetShareReference(name).Exists()
}

func (d *Driver) getFileSvcClient(accountName, accountKey string) (*azs.FileServiceClient, error) {
	fileClient, err := azs.NewClient(accountName, accountKey, d.cloud.Environment.StorageEndpointSuffix, azs.DefaultAPIVersion, true)
	if err != nil {
		return nil, fmt.Errorf("error creating azure client: %v", err)
	}
	fc := fileClient.GetFileService()
	return &fc, nil
}

// get file share info according to volume id, e.g.
// input: "rg#f5713de20cde511e8ba4900#pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41#diskname.vhd"
// output: rg, f5713de20cde511e8ba4900, pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41, diskname.vhd
func getFileShareInfo(id string) (string, string, string, string, error) {
	segments := strings.Split(id, separator)
	if len(segments) < 3 {
		return "", "", "", "", fmt.Errorf("error parsing volume id: %q, should at least contain two #", id)
	}
	var diskName string
	if len(segments) > 3 {
		diskName = segments[3]
	}
	return segments[0], segments[1], segments[2], diskName, nil
}

// check whether mountOptions contains file_mode, dir_mode, vers, if not, append default mode
func appendDefaultMountOptions(mountOptions []string) []string {
	fileModeFlag := false
	dirModeFlag := false
	versFlag := false

	for _, mountOption := range mountOptions {
		if strings.HasPrefix(mountOption, fileMode) {
			fileModeFlag = true
		}
		if strings.HasPrefix(mountOption, dirMode) {
			dirModeFlag = true
		}
		if strings.HasPrefix(mountOption, vers) {
			versFlag = true
		}
	}

	allMountOptions := mountOptions
	if !fileModeFlag {
		allMountOptions = append(allMountOptions, fmt.Sprintf("%s=%s", fileMode, defaultFileMode))
	}

	if !dirModeFlag {
		allMountOptions = append(allMountOptions, fmt.Sprintf("%s=%s", dirMode, defaultDirMode))
	}

	if !versFlag {
		allMountOptions = append(allMountOptions, fmt.Sprintf("%s=%s", vers, defaultVers))
	}

	/* todo: looks like fsGroup is not included in CSI
	if !gidFlag && fsGroup != nil {
		allMountOptions = append(allMountOptions, fmt.Sprintf("%s=%d", gid, *fsGroup))
	}
	*/
	return allMountOptions
}

// get storage account from secrets map
func getStorageAccount(secrets map[string]string) (string, string, error) {
	if secrets == nil {
		return "", "", fmt.Errorf("unexpected: getStorageAccount secrets is nil")
	}

	var accountName, accountKey string
	for k, v := range secrets {
		switch strings.ToLower(k) {
		case "accountname":
			accountName = v
		case "azurestorageaccountname": // for compatibility with built-in smb plugin
			accountName = v
		case "accountkey":
			accountKey = v
		case "azurestorageaccountkey": // for compatibility with built-in smb plugin
			accountKey = v
		}
	}

	if accountName == "" {
		return "", "", fmt.Errorf("could not find accountname or azurestorageaccountname field secrets(%v)", secrets)
	}
	if accountKey == "" {
		return "", "", fmt.Errorf("could not find accountkey or azurestorageaccountkey field in secrets(%v)", secrets)
	}

	return accountName, accountKey, nil
}

// File share names can contain only lowercase letters, numbers, and hyphens,
// and must begin and end with a letter or a number,
// and must be from 3 through 63 characters long.
// The name cannot contain two consecutive hyphens.
//
// See https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-shares--directories--files--and-metadata#share-names
func getValidFileShareName(volumeName string) string {
	fileShareName := strings.ToLower(volumeName)
	if len(fileShareName) > fileShareNameMaxLength {
		fileShareName = fileShareName[0:fileShareNameMaxLength]
	}
	if !checkShareNameBeginAndEnd(fileShareName) || len(fileShareName) < fileShareNameMinLength {
		fileShareName = util.GenerateVolumeName("pvc-file", uuid.NewUUID().String(), fileShareNameMaxLength)
		klog.Warningf("the requested volume name (%q) is invalid, so it is regenerated as (%q)", volumeName, fileShareName)
	}
	fileShareName = strings.Replace(fileShareName, "--", "-", -1)

	return fileShareName
}

func checkShareNameBeginAndEnd(fileShareName string) bool {
	length := len(fileShareName)
	if (('a' <= fileShareName[0] && fileShareName[0] <= 'z') ||
		('0' <= fileShareName[0] && fileShareName[0] <= '9')) &&
		(('a' <= fileShareName[length-1] && fileShareName[length-1] <= 'z') ||
			('0' <= fileShareName[length-1] && fileShareName[length-1] <= '9')) {
		return true
	}

	return false
}

// get snapshot name according to snapshot id, e.g.
// input: "rg#f5713de20cde511e8ba4900#csivolumename#diskname#2019-08-22T07:17:53.0000000Z"
// output: 2019-08-22T07:17:53.0000000Z
func getSnapshot(id string) (string, error) {
	segments := strings.Split(id, separator)
	if len(segments) < 5 {
		return "", fmt.Errorf("error parsing volume id: %q, should at least contain four #", id)
	}
	return segments[4], nil
}

func getFileURL(accountName, accountKey, storageEndpointSuffix, fileShareName, diskName string) (*azfile.FileURL, error) {
	credential, err := azfile.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("NewSharedKeyCredential(%s) failed with error: %v", accountName, err)
	}
	u, err := url.Parse(fmt.Sprintf(fileURLTemplate, accountName, storageEndpointSuffix, fileShareName, diskName))
	if err != nil {
		return nil, fmt.Errorf("parse fileURLTemplate error: %v", err)
	}
	if u == nil {
		return nil, fmt.Errorf("parse fileURLTemplate error: url is nil")
	}
	po := azfile.PipelineOptions{
		// Set RetryOptions to control how HTTP request are retried when retryable failures occur
		Retry: azfile.RetryOptions{
			Policy:        azfile.RetryPolicyExponential, // Use exponential backoff as opposed to linear
			MaxTries:      3,                             // Try at most 3 times to perform the operation (set to 1 to disable retries)
			TryTimeout:    time.Second * 3,               // Maximum time allowed for any single try
			RetryDelay:    time.Second * 1,               // Backoff amount for each retry (exponential or linear)
			MaxRetryDelay: time.Second * 3,               // Max delay between retries
		},
	}
	fileURL := azfile.NewFileURL(*u, azfile.NewPipeline(credential, po))
	return &fileURL, nil
}

func createDisk(ctx context.Context, accountName, accountKey, storageEndpointSuffix, fileShareName, diskName string, diskSizeBytes int64) error {
	vhdHeader := vhd.CreateFixedHeader(uint64(diskSizeBytes), &vhd.VHDOptions{})
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, vhdHeader); nil != err {
		return fmt.Errorf("failed to write VHDHeader(%+v): %v", vhdHeader, err)
	}
	headerBytes := buf.Bytes()
	start := diskSizeBytes - int64(len(headerBytes))
	end := diskSizeBytes - 1

	fileURL, err := getFileURL(accountName, accountKey, storageEndpointSuffix, fileShareName, diskName)
	if err != nil {
		return err
	}
	if fileURL == nil {
		return fmt.Errorf("getFileURL(%s,%s,%s,%s) return empty fileURL", accountName, storageEndpointSuffix, fileShareName, diskName)
	}
	if _, err = fileURL.Create(ctx, diskSizeBytes, azfile.FileHTTPHeaders{}, azfile.Metadata{}); err != nil {
		return err
	}
	if _, err = fileURL.UploadRange(ctx, end-start, bytes.NewReader(headerBytes[:vhd.VHD_HEADER_SIZE]), nil); err != nil {
		return err
	}
	return nil
}

func IsCorruptedDir(dir string) bool {
	_, pathErr := mount.PathExists(dir)
	fmt.Printf("IsCorruptedDir(%s) returned with error: %v", dir, pathErr)
	return pathErr != nil && mount.IsCorruptedMnt(pathErr)
}

func (d *Driver) GetAccountInfo(volumeID string, secrets, reqContext map[string]string) (rgName, accountName, accountKey, fileShareName, diskName string, err error) {
	if len(secrets) == 0 {
		rgName, accountName, fileShareName, diskName, err = getFileShareInfo(volumeID)
		if err == nil {
			if rgName == "" {
				rgName = d.cloud.ResourceGroup
			}
			if d.cloud.KubeClient != nil {
				secretName := fmt.Sprintf(secretNameTemplate, accountName)
				secretNamespace := reqContext[secretNamespaceField]
				if secretNamespace == "" {
					secretNamespace = defaultSecretNamespace
				}
				secret, err := d.cloud.KubeClient.CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
				if err != nil {
					klog.V(4).Infof("could not get secret(%v): %v", secretName, err)
				} else {
					accountKey = string(secret.Data[defaultSecretAccountKey][:])
				}
			}
			if accountKey == "" {
				accountKey, err = d.cloud.GetStorageAccesskey(accountName, rgName)
			}
		}
	} else {
		for k, v := range reqContext {
			switch strings.ToLower(k) {
			case shareNameField:
				fileShareName = v
			case diskNameField:
				diskName = v
			}
		}
		if fileShareName != "" {
			accountName, accountKey, err = getStorageAccount(secrets)
		} else {
			err = fmt.Errorf("could not find sharename from context(%v)", reqContext)
		}
	}

	return rgName, accountName, accountKey, fileShareName, diskName, err
}
