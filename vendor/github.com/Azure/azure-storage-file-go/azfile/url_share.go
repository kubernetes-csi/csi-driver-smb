package azfile

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

// A ShareURL represents a URL to the Azure Storage share allowing you to manipulate its directories and files.
type ShareURL struct {
	shareClient shareClient
}

// NewShareURL creates a ShareURL object using the specified URL and request policy pipeline.
// Note: p can't be nil.
func NewShareURL(url url.URL, p pipeline.Pipeline) ShareURL {
	shareClient := newShareClient(url, p)
	return ShareURL{shareClient: shareClient}
}

// URL returns the URL endpoint used by the ShareURL object.
func (s ShareURL) URL() url.URL {
	return s.shareClient.URL()
}

// String returns the URL as a string.
func (s ShareURL) String() string {
	u := s.URL()
	return u.String()
}

// WithPipeline creates a new ShareURL object identical to the source but with the specified request policy pipeline.
func (s ShareURL) WithPipeline(p pipeline.Pipeline) ShareURL {
	return NewShareURL(s.URL(), p)
}

// WithSnapshot creates a new ShareURL object identical to the source but with the specified snapshot timestamp.
// Pass time.Time{} to remove the snapshot returning a URL to the base share.
func (s ShareURL) WithSnapshot(snapshot string) ShareURL {
	p := NewFileURLParts(s.URL())
	p.ShareSnapshot = snapshot
	return NewShareURL(p.URL(), s.shareClient.Pipeline())
}

// NewDirectoryURL creates a new DirectoryURL object by concatenating directoryName to the end of
// ShareURL's URL. The new DirectoryURL uses the same request policy pipeline as the ShareURL.
// To change the pipeline, create the DirectoryURL and then call its WithPipeline method passing in the
// desired pipeline object. Or, call this package's NewDirectoryURL instead of calling this object's
// NewDirectoryURL method.
func (s ShareURL) NewDirectoryURL(directoryName string) DirectoryURL {
	directoryURL := appendToURLPath(s.URL(), directoryName)
	return NewDirectoryURL(directoryURL, s.shareClient.Pipeline())
}

// NewRootDirectoryURL creates a new DirectoryURL object using ShareURL's URL.
// The new DirectoryURL uses the same request policy pipeline as the
// ShareURL. To change the pipeline, create the DirectoryURL and then call its WithPipeline method
// passing in the desired pipeline object. Or, call NewDirectoryURL instead of calling the NewDirectoryURL method.
func (s ShareURL) NewRootDirectoryURL() DirectoryURL {
	return NewDirectoryURL(s.URL(), s.shareClient.Pipeline())
}

// Create creates a new share within a storage account. If a share with the same name already exists, the operation fails.
// quotaInGB specifies the maximum size of the share in gigabytes, 0 means you accept service's default quota.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/create-share.
func (s ShareURL) Create(ctx context.Context, metadata Metadata, quotaInGB int32) (*ShareCreateResponse, error) {
	var quota *int32
	if quotaInGB != 0 {
		quota = &quotaInGB
	}
	return s.shareClient.Create(ctx, nil, metadata, quota)
}

// CreateSnapshot creates a read-only snapshot of a share.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/snapshot-share.
func (s ShareURL) CreateSnapshot(ctx context.Context, metadata Metadata) (*ShareCreateSnapshotResponse, error) {
	return s.shareClient.CreateSnapshot(ctx, nil, metadata)
}

// Delete marks the specified share or share snapshot for deletion.
// The share or share snapshot and any files contained within it are later deleted during garbage collection.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/delete-share.
func (s ShareURL) Delete(ctx context.Context, deleteSnapshotsOption DeleteSnapshotsOptionType) (*ShareDeleteResponse, error) {
	return s.shareClient.Delete(ctx, nil, nil, deleteSnapshotsOption)
}

// GetProperties returns all user-defined metadata and system properties for the specified share or share snapshot.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/get-share-properties.
func (s ShareURL) GetProperties(ctx context.Context) (*ShareGetPropertiesResponse, error) {
	return s.shareClient.GetProperties(ctx, nil, nil)
}

// SetQuota sets service-defined properties for the specified share.
// quotaInGB specifies the maximum size of the share in gigabytes, 0 means no quote and uses service's default value.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/set-share-properties.
func (s ShareURL) SetQuota(ctx context.Context, quotaInGB int32) (*ShareSetQuotaResponse, error) {
	var quota *int32
	if quotaInGB != 0 {
		quota = &quotaInGB
	}
	return s.shareClient.SetQuota(ctx, nil, quota)
}

// SetMetadata sets the share's metadata.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/set-share-metadata.
func (s ShareURL) SetMetadata(ctx context.Context, metadata Metadata) (*ShareSetMetadataResponse, error) {
	return s.shareClient.SetMetadata(ctx, nil, metadata)
}

// GetPermissions returns information about stored access policies specified on the share.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/get-share-acl.
func (s ShareURL) GetPermissions(ctx context.Context) (*SignedIdentifiers, error) {
	return s.shareClient.GetAccessPolicy(ctx, nil)
}

// CreatePermission uploads a SDDL permission string, and returns a permission key to use in conjunction with a file or folder.
// Note that this is only required for 9KB or larger permission strings.
// Furthermore, note that SDDL strings should be converted to a portable format before being uploaded.
// In order to make a SDDL portable, please replace well-known SIDs with their domain specific counterpart.
// Well-known SIDs are listed here: https://docs.microsoft.com/en-us/windows/win32/secauthz/sid-strings
// More info about SDDL strings can be located at: https://docs.microsoft.com/en-us/windows/win32/secauthz/security-descriptor-string-format
func (s ShareURL) CreatePermission(ctx context.Context, permission string) (*ShareCreatePermissionResponse, error) {
	perm := SharePermission{Permission: permission}
	return s.shareClient.CreatePermission(ctx, perm, nil)
}

// GetPermission obtains a SDDL permission string from the service using a known permission key.
func (s ShareURL) GetPermission(ctx context.Context, permissionKey string) (*SharePermission, error) {
	return s.shareClient.GetPermission(ctx, permissionKey, nil)
}

// The AccessPolicyPermission type simplifies creating the permissions string for a share's access policy.
// Initialize an instance of this type and then call its String method to set AccessPolicy's Permission field.
type AccessPolicyPermission struct {
	Read, Create, Write, Delete, List bool
}

// String produces the access policy permission string for an Azure Storage share.
// Call this method to set AccessPolicy's Permission field.
func (p AccessPolicyPermission) String() string {
	var b bytes.Buffer
	if p.Read {
		b.WriteRune('r')
	}
	if p.Create {
		b.WriteRune('c')
	}
	if p.Write {
		b.WriteRune('w')
	}
	if p.Delete {
		b.WriteRune('d')
	}
	if p.List {
		b.WriteRune('l')
	}
	return b.String()
}

// Parse initializes the AccessPolicyPermission's fields from a string.
func (p *AccessPolicyPermission) Parse(s string) {
	p.Read = strings.ContainsRune(s, 'r')
	p.Create = strings.ContainsRune(s, 'c')
	p.Write = strings.ContainsRune(s, 'w')
	p.Delete = strings.ContainsRune(s, 'd')
	p.List = strings.ContainsRune(s, 'l')
}

// SetPermissions sets a stored access policy for use with shared access signatures.
// For more information, see https://docs.microsoft.com/rest/api/storageservices/set-share-acl.
func (s ShareURL) SetPermissions(ctx context.Context, permissions []SignedIdentifier) (*ShareSetAccessPolicyResponse, error) {
	return s.shareClient.SetAccessPolicy(ctx, permissions, nil)
}

// GetStatistics retrieves statistics related to the share.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/get-share-stats.
func (s ShareURL) GetStatistics(ctx context.Context) (*ShareStats, error) {
	return s.shareClient.GetStatistics(ctx, nil)
}
