package azfile

// https://docs.microsoft.com/en-us/rest/api/storageservices/file-service-error-codes

// ServiceCode values indicate a service failure.
const (
	// The file or directory could not be deleted because it is in use by an SMB client (409).
	ServiceCodeCannotDeleteFileOrDirectory ServiceCodeType = "CannotDeleteFileOrDirectory"

	// The specified resource state could not be flushed from an SMB client in the specified time (500).
	ServiceCodeClientCacheFlushDelay ServiceCodeType = "ClientCacheFlushDelay"

	// The specified resource is marked for deletion by an SMB client (409).
	ServiceCodeDeletePending ServiceCodeType = "DeletePending"

	// The specified directory is not empty (409).
	ServiceCodeDirectoryNotEmpty ServiceCodeType = "DirectoryNotEmpty"

	// A portion of the specified file is locked by an SMB client (409).
	ServiceCodeFileLockConflict ServiceCodeType = "FileLockConflict"

	// File or directory path is too long (400).
	// Or File or directory path has too many subdirectories (400).
	ServiceCodeInvalidFileOrDirectoryPathName ServiceCodeType = "InvalidFileOrDirectoryPathName"

	// The specified parent path does not exist (404).
	ServiceCodeParentNotFound ServiceCodeType = "ParentNotFound"

	// The specified resource is read-only and cannot be modified at this time (409).
	ServiceCodeReadOnlyAttribute ServiceCodeType = "ReadOnlyAttribute"

	// The specified share already exists (409).
	ServiceCodeShareAlreadyExists ServiceCodeType = "ShareAlreadyExists"

	// The specified share is being deleted. Try operation later (409).
	ServiceCodeShareBeingDeleted ServiceCodeType = "ShareBeingDeleted"

	// The specified share is disabled by the administrator (403).
	ServiceCodeShareDisabled ServiceCodeType = "ShareDisabled"

	// The specified share does not exist (404).
	ServiceCodeShareNotFound ServiceCodeType = "ShareNotFound"

	// The specified resource may be in use by an SMB client (409).
	ServiceCodeSharingViolation ServiceCodeType = "SharingViolation"

	// Another Share Snapshot operation is in progress (409).
	ServiceCodeShareSnapshotInProgress ServiceCodeType = "ShareSnapshotInProgress"

	// The total number of snapshots for the share is over the limit (409).
	ServiceCodeShareSnapshotCountExceeded ServiceCodeType = "ShareSnapshotCountExceeded"

	// The operation is not supported on a share snapshot (400).
	ServiceCodeShareSnapshotOperationNotSupported ServiceCodeType = "ShareSnapshotOperationNotSupported"

	// The share has snapshots and the operation requires no snapshots (409).
	ServiceCodeShareHasSnapshots ServiceCodeType = "ShareHasSnapshots"
)
