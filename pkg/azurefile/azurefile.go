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

package azurefile

import (
	"fmt"
	"strings"

	csicommon "sigs.k8s.io/azurefile-csi-driver/pkg/csi-common"
	volumehelper "sigs.k8s.io/azurefile-csi-driver/pkg/util"

	azs "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume/util"
	"k8s.io/legacy-cloud-providers/azure"
)

const (
	DriverName       = "file.csi.azure.com"
	separator        = "#"
	volumeIDTemplate = "%s#%s#%s"
	fileURLTemplate  = "https://%s.file.%s"
	fileMode         = "file_mode"
	dirMode          = "dir_mode"
	vers             = "vers"
	defaultFileMode  = "0777"
	defaultDirMode   = "0777"
	defaultVers      = "3.0"

	// See https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-shares--directories--files--and-metadata#share-names
	fileShareNameMinLength = 3
	fileShareNameMaxLength = 63

	// Minimum size of Azure Premium Files is 100GiB
	// See https://docs.microsoft.com/en-us/azure/storage/files/storage-files-planning#provisioned-shares
	defaultAzureFileQuota = 100

	// key of snapshot name in metadata
	snapshotNameKey = "initiator"
)

// Driver implements all interfaces of CSI drivers
type Driver struct {
	csicommon.CSIDriver
	cloud   *azure.Cloud
	mounter *mount.SafeFormatAndMount
}

// NewDriver Creates a NewCSIDriver object. Assumes vendor version is equal to driver version &
// does not support optional driver plugin info manifest field. Refer to CSI spec for more details.
func NewDriver(nodeID string) *Driver {
	driver := Driver{}
	driver.Name = DriverName
	driver.Version = driverVersion
	driver.NodeID = nodeID
	return &driver
}

// Run driver initialization
func (d *Driver) Run(endpoint string) {
	versionMeta, err := GetVersionYAML()
	if err != nil {
		klog.Fatalf("%v", err)
	}
	klog.Infof("\nDRIVER INFORMATION:\n-------------------\n%s\n\nStreaming logs below:", versionMeta)

	cloud, err := GetCloudProvider()
	if err != nil {
		klog.Fatalln("failed to get Azure Cloud Provider")
	}
	d.cloud = cloud

	d.mounter = &mount.SafeFormatAndMount{
		Interface: mount.New(""),
		Exec:      mount.NewOsExec(),
	}

	// Initialize default library driver
	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
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
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
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
// input: "rg#f5713de20cde511e8ba4900#pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41"
// output: rg, f5713de20cde511e8ba4900, pvc-file-dynamic-17e43f84-f474-11e8-acd0-000d3a00df41
func getFileShareInfo(id string) (string, string, string, error) {
	segments := strings.Split(id, separator)
	if len(segments) < 3 {
		return "", "", "", fmt.Errorf("error parsing volume id: %q, should at least contain two #", id)
	}
	return segments[0], segments[1], segments[2], nil
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
		case "azurestorageaccountname": // for compatibility with built-in azurefile plugin
			accountName = v
		case "accountkey":
			accountKey = v
		case "azurestorageaccountkey": // for compatibility with built-in azurefile plugin
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
// input: "rg#f5713de20cde511e8ba4900#csivolumename#2019-08-22T07:17:53.0000000Z"
// output: 2019-08-22T07:17:53.0000000Z
func getSnapshot(id string) (string, error) {
	segments := strings.Split(id, separator)
	if len(segments) != 4 {
		return "", fmt.Errorf("error parsing volume id: %q, should at least contain three #", id)
	}
	return segments[3], nil
}

func (d *Driver) expandVolume(ctx context.Context, volumeID string, capacityBytes int64) (int64, error) {
	if capacityBytes == 0 {
		return -1, status.Error(codes.InvalidArgument, "volume capacity range missing in request")
	}
	requestGiB := int32(volumehelper.RoundUpGiB(capacityBytes))

	shareURL, err := d.getShareURL(volumeID)
	if err != nil {
		return -1, status.Errorf(codes.Internal, "failed to get share url with (%s): %v, returning with success", volumeID, err)
	}

	if _, err = shareURL.SetQuota(ctx, requestGiB); err != nil {
		return -1, status.Errorf(codes.Internal, "expand volume error: %v", err)
	}

	resp, err := shareURL.GetProperties(ctx)
	if err != nil {
		return -1, status.Errorf(codes.Internal, "failed to get properties of share(%v): %v", shareURL, err)
	}

	return volumehelper.GiBToBytes(int64(resp.Quota())), nil
}
