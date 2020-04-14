package azfile

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
	"time"
)

// FileHTTPHeaders contains read/writeable file properties.
type FileHTTPHeaders struct {
	ContentType        string
	ContentMD5         []byte
	ContentEncoding    string
	ContentLanguage    string
	ContentDisposition string
	CacheControl       string

	SMBProperties
}

// FileCreationTime and FileLastWriteTime are handled by the Azure Files service with high-precision ISO-8601 timestamps.
// Use this format to parse these fields and format them.
const ISO8601 = "2006-01-02T15:04:05.0000000Z"   // must have 0's for fractional seconds, because Files Service requires fixed width

// SMBPropertyHolder is an interface designed for SMBPropertyAdapter, to identify valid response types for adapting.
type SMBPropertyHolder interface {
	FileCreationTime() string
	FileLastWriteTime() string
	FileAttributes() string
}

// SMBPropertyAdapter is a wrapper struct that automatically converts the string outputs of FileAttributes, FileCreationTime and FileLastWrite time to time.Time.
// It is _not_ error resistant. It is expected that the response you're inserting into this is a valid response.
// File and directory calls that return such properties are: GetProperties, SetProperties, Create
// File Downloads also return such properties. Insert other response types at your peril.
type SMBPropertyAdapter struct {
	PropertySource SMBPropertyHolder
}

func (s *SMBPropertyAdapter) convertISO8601(input string) time.Time {
	t, err := time.Parse(ISO8601, input)

	if err != nil {
		// This should literally never happen if this struct is used correctly.
		panic("SMBPropertyAdapter expects a successful response fitting the SMBPropertyHolder interface. Failed to parse time:\n" + err.Error())
	}

	return t
}

func (s *SMBPropertyAdapter) FileCreationTime() time.Time {
	return s.convertISO8601(s.PropertySource.FileCreationTime()).UTC()
}

func (s *SMBPropertyAdapter) FileLastWriteTime() time.Time {
	return s.convertISO8601(s.PropertySource.FileLastWriteTime()).UTC()
}

func (s *SMBPropertyAdapter) FileAttributes() FileAttributeFlags {
	return ParseFileAttributeFlagsString(s.PropertySource.FileAttributes())
}

// SMBProperties defines a struct that takes in optional parameters regarding SMB/NTFS properties.
// When you pass this into another function (Either literally or via FileHTTPHeaders), the response will probably fit inside SMBPropertyAdapter.
// Nil values of the properties are inferred to be preserved (or when creating, use defaults). Clearing a value can be done by supplying an empty item instead of nil.
type SMBProperties struct {
	// NOTE: If pointers are nil, we infer that you wish to preserve these properties. To clear them, point to an empty string.
	// NOTE: Permission strings are required to be sub-9KB. Please upload the permission to the share, and submit a key instead if yours exceeds this limit.
	PermissionString *string
	PermissionKey *string
	// In Windows, a 32 bit file attributes integer exists. This is that.
	FileAttributes *FileAttributeFlags
	// A UTC time-date string is specified below. A value of 'now' defaults to now. 'preserve' defaults to preserving the old case.
	FileCreationTime *time.Time
	FileLastWriteTime *time.Time
}

// SetISO8601CreationTime sets the file creation time with a string formatted as ISO8601
func (sp *SMBProperties) SetISO8601CreationTime(input string) error {
	t, err := time.Parse(ISO8601, input)

	if err != nil {
		return err
	}

	sp.FileCreationTime = &t
	return nil
}

// SetISO8601WriteTime sets the file last write time with a string formatted as ISO8601
func (sp *SMBProperties) SetISO8601WriteTime(input string) error {
	t, err := time.Parse(ISO8601, input)

	if err != nil {
		return err
	}

	sp.FileLastWriteTime = &t
	return nil
}

func (sp *SMBProperties) selectSMBPropertyValues(isDir bool, defaultPerm, defaultAttribs, defaultTime string) (permStr, permKey *string, attribs, creationTime, lastWriteTime string, err error) {
	permStr = &defaultPerm
	if sp.PermissionString != nil {
		permStr = sp.PermissionString
	}

	if sp.PermissionKey != nil {
		if permStr == &defaultPerm {
			permStr = nil
		} else if permStr != nil {
			err = errors.New("only permission string OR permission key may be used")
			return
		}

		permKey = sp.PermissionKey
	}

	attribs = defaultAttribs
	if sp.FileAttributes != nil {
		attribs = sp.FileAttributes.String()
		if isDir && strings.ToLower(attribs) != "none"  {   // must test string, not sp.FileAttributes, since it may contain set bits that we don't convert
			// Directories need to have this attribute included, if setting any attributes.
			// We don't expose it in FileAttributes because it doesn't do anything useful to consumers of
			// this SDK. And because it always needs to be set for directories and not for non-directories,
			// so it makes sense to automate that here.
			attribs += "|Directory"
		}
	}

	creationTime = defaultTime
	if sp.FileCreationTime != nil {
		creationTime = sp.FileCreationTime.UTC().Format(ISO8601)
	}

	lastWriteTime = defaultTime
	if sp.FileLastWriteTime != nil {
		lastWriteTime = sp.FileLastWriteTime.UTC().Format(ISO8601)
	}

	return
}

// NewHTTPHeaders returns the user-modifiable properties for this file.
func (dr RetryableDownloadResponse) NewHTTPHeaders() FileHTTPHeaders {
	return FileHTTPHeaders{
		ContentType:        dr.ContentType(),
		ContentEncoding:    dr.ContentEncoding(),
		ContentLanguage:    dr.ContentLanguage(),
		ContentDisposition: dr.ContentDisposition(),
		CacheControl:       dr.CacheControl(),
		ContentMD5:         dr.ContentMD5(),
	}
}

// NewHTTPHeaders returns the user-modifiable properties for this file.
func (fgpr FileGetPropertiesResponse) NewHTTPHeaders() FileHTTPHeaders {
	return FileHTTPHeaders{
		ContentType:        fgpr.ContentType(),
		ContentEncoding:    fgpr.ContentEncoding(),
		ContentLanguage:    fgpr.ContentLanguage(),
		ContentDisposition: fgpr.ContentDisposition(),
		CacheControl:       fgpr.CacheControl(),
		ContentMD5:         fgpr.ContentMD5(),
	}
}

// RetryableDownloadResponse wraps AutoRest generated DownloadResponse and helps to provide info for retry.
type RetryableDownloadResponse struct {
	dr *DownloadResponse

	// Fields need for retry.
	ctx  context.Context
	f    FileURL
	info HTTPGetterInfo
}

// Response returns the raw HTTP response object.
func (dr RetryableDownloadResponse) Response() *http.Response {
	return dr.dr.Response()
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (dr RetryableDownloadResponse) StatusCode() int {
	return dr.dr.StatusCode()
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (dr RetryableDownloadResponse) Status() string {
	return dr.dr.Status()
}

// AcceptRanges returns the value for header Accept-Ranges.
func (dr RetryableDownloadResponse) AcceptRanges() string {
	return dr.dr.AcceptRanges()
}

// CacheControl returns the value for header Cache-Control.
func (dr RetryableDownloadResponse) CacheControl() string {
	return dr.dr.CacheControl()
}

// ContentDisposition returns the value for header Content-Disposition.
func (dr RetryableDownloadResponse) ContentDisposition() string {
	return dr.dr.ContentDisposition()
}

// ContentEncoding returns the value for header Content-Encoding.
func (dr RetryableDownloadResponse) ContentEncoding() string {
	return dr.dr.ContentEncoding()
}

// ContentLanguage returns the value for header Content-Language.
func (dr RetryableDownloadResponse) ContentLanguage() string {
	return dr.dr.ContentLanguage()
}

// ContentLength returns the value for header Content-Length.
func (dr RetryableDownloadResponse) ContentLength() int64 {
	return dr.dr.ContentLength()
}

// ContentRange returns the value for header Content-Range.
func (dr RetryableDownloadResponse) ContentRange() string {
	return dr.dr.ContentRange()
}

// ContentType returns the value for header Content-Type.
func (dr RetryableDownloadResponse) ContentType() string {
	return dr.dr.ContentType()
}

// CopyCompletionTime returns the value for header x-ms-copy-completion-time.
func (dr RetryableDownloadResponse) CopyCompletionTime() time.Time {
	return dr.dr.CopyCompletionTime()
}

// CopyID returns the value for header x-ms-copy-id.
func (dr RetryableDownloadResponse) CopyID() string {
	return dr.dr.CopyID()
}

// CopyProgress returns the value for header x-ms-copy-progress.
func (dr RetryableDownloadResponse) CopyProgress() string {
	return dr.dr.CopyProgress()
}

// CopySource returns the value for header x-ms-copy-source.
func (dr RetryableDownloadResponse) CopySource() string {
	return dr.dr.CopySource()
}

// CopyStatus returns the value for header x-ms-copy-status.
func (dr RetryableDownloadResponse) CopyStatus() CopyStatusType {
	return dr.dr.CopyStatus()
}

// CopyStatusDescription returns the value for header x-ms-copy-status-description.
func (dr RetryableDownloadResponse) CopyStatusDescription() string {
	return dr.dr.CopyStatusDescription()
}

// Date returns the value for header Date.
func (dr RetryableDownloadResponse) Date() time.Time {
	return dr.dr.Date()
}

// ETag returns the value for header ETag.
func (dr RetryableDownloadResponse) ETag() ETag {
	return dr.dr.ETag()
}

// IsServerEncrypted returns the value for header x-ms-server-encrypted.
func (dr RetryableDownloadResponse) IsServerEncrypted() string {
	return dr.dr.IsServerEncrypted()
}

// LastModified returns the value for header Last-Modified.
func (dr RetryableDownloadResponse) LastModified() time.Time {
	return dr.dr.LastModified()
}

// RequestID returns the value for header x-ms-request-id.
func (dr RetryableDownloadResponse) RequestID() string {
	return dr.dr.RequestID()
}

// Version returns the value for header x-ms-version.
func (dr RetryableDownloadResponse) Version() string {
	return dr.dr.Version()
}

// NewMetadata returns user-defined key/value pairs.
func (dr RetryableDownloadResponse) NewMetadata() Metadata {
	return dr.dr.NewMetadata()
}

// FileContentMD5 returns the value for header x-ms-content-md5.
func (dr RetryableDownloadResponse) FileContentMD5() []byte {
	return dr.dr.FileContentMD5()
}

// ContentMD5 returns the value for header Content-MD5.
func (dr RetryableDownloadResponse) ContentMD5() []byte {
	return dr.dr.ContentMD5()
}

// FileItem - Listed file item.
type FileItem struct {
	// XMLName is used for marshalling and is subject to removal in a future release.
	XMLName xml.Name `xml:"File"`
	// Name - Name of the entry.
	Name       string        `xml:"Name"`
	Properties *FileProperty `xml:"Properties"`
}

// DirectoryItem - Listed directory item.
type DirectoryItem struct {
	// XMLName is used for marshalling and is subject to removal in a future release.
	XMLName xml.Name `xml:"Directory"`
	// Name - Name of the entry.
	Name string `xml:"Name"`
}

// ListFilesAndDirectoriesSegmentResponse - An enumeration of directories and files.
type ListFilesAndDirectoriesSegmentResponse struct {
	rawResponse *http.Response
	// XMLName is used for marshalling and is subject to removal in a future release.
	XMLName         xml.Name        `xml:"EnumerationResults"`
	ServiceEndpoint string          `xml:"ServiceEndpoint,attr"`
	ShareName       string          `xml:"ShareName,attr"`
	ShareSnapshot   *string         `xml:"ShareSnapshot,attr"`
	DirectoryPath   string          `xml:"DirectoryPath,attr"`
	Prefix          string          `xml:"Prefix"`
	Marker          *string         `xml:"Marker"`
	MaxResults      *int32          `xml:"MaxResults"`
	FileItems       []FileItem      `xml:"Entries>File"`
	DirectoryItems  []DirectoryItem `xml:"Entries>Directory"`
	NextMarker      Marker          `xml:"NextMarker"`
}

// Response returns the raw HTTP response object.
func (ldafr ListFilesAndDirectoriesSegmentResponse) Response() *http.Response {
	return ldafr.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (ldafr ListFilesAndDirectoriesSegmentResponse) StatusCode() int {
	return ldafr.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (ldafr ListFilesAndDirectoriesSegmentResponse) Status() string {
	return ldafr.rawResponse.Status
}

// ContentType returns the value for header Content-Type.
func (ldafr ListFilesAndDirectoriesSegmentResponse) ContentType() string {
	return ldafr.rawResponse.Header.Get("Content-Type")
}

// Date returns the value for header Date.
func (ldafr ListFilesAndDirectoriesSegmentResponse) Date() time.Time {
	s := ldafr.rawResponse.Header.Get("Date")
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		panic(err)
	}
	return t
}

// RequestID returns the value for header x-ms-request-id.
func (ldafr ListFilesAndDirectoriesSegmentResponse) RequestID() string {
	return ldafr.rawResponse.Header.Get("x-ms-request-id")
}

// Version returns the value for header x-ms-version.
func (ldafr ListFilesAndDirectoriesSegmentResponse) Version() string {
	return ldafr.rawResponse.Header.Get("x-ms-version")
}

// ErrorCode returns the value for header x-ms-error-code.
func (ldafr ListFilesAndDirectoriesSegmentResponse) ErrorCode() string {
	return ldafr.rawResponse.Header.Get("x-ms-error-code")
}

// MetricProperties definies convenience struct for Metrics,
type MetricProperties struct {
	// MetricEnabled - Indicates whether metrics are enabled for the File service.
	MetricEnabled bool
	// Version - The version of Storage Analytics to configure.
	// Version string, comment out version, as it's mandatory and should be 1.0
	// IncludeAPIs - Indicates whether metrics should generate summary statistics for called API operations.
	IncludeAPIs bool
	// RetentionPolicyEnabled - Indicates whether a rentention policy is enabled for the File service.
	RetentionPolicyEnabled bool
	// RetentionDays - Indicates the number of days that metrics data should be retained.
	RetentionDays int32
}

// FileServiceProperties defines convenience struct for StorageServiceProperties
type FileServiceProperties struct {
	rawResponse *http.Response
	// HourMetrics - A summary of request statistics grouped by API in hourly aggregates for files.
	HourMetrics MetricProperties
	// MinuteMetrics - A summary of request statistics grouped by API in minute aggregates for files.
	MinuteMetrics MetricProperties
	// Cors - The set of CORS rules.
	Cors []CorsRule
}

// Response returns the raw HTTP response object.
func (fsp FileServiceProperties) Response() *http.Response {
	return fsp.rawResponse
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (fsp FileServiceProperties) StatusCode() int {
	return fsp.rawResponse.StatusCode
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (fsp FileServiceProperties) Status() string {
	return fsp.rawResponse.Status
}

// RequestID returns the value for header x-ms-request-id.
func (fsp FileServiceProperties) RequestID() string {
	return fsp.rawResponse.Header.Get("x-ms-request-id")
}

// Version returns the value for header x-ms-version.
func (fsp FileServiceProperties) Version() string {
	return fsp.rawResponse.Header.Get("x-ms-version")
}
