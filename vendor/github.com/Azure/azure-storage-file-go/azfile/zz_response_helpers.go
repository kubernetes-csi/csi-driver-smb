package azfile

import (
	"context"
	"encoding/xml"
	"net/http"
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
}

// NewHTTPHeaders returns the user-modifiable properties for this file.
func (dr DownloadResponse) NewHTTPHeaders() FileHTTPHeaders {
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

// DownloadResponse wraps AutoRest generated downloadResponse and helps to provide info for retry.
type DownloadResponse struct {
	dr *downloadResponse

	// Fields need for retry.
	ctx  context.Context
	f    FileURL
	info HTTPGetterInfo
}

// Response returns the raw HTTP response object.
func (dr DownloadResponse) Response() *http.Response {
	return dr.dr.Response()
}

// StatusCode returns the HTTP status code of the response, e.g. 200.
func (dr DownloadResponse) StatusCode() int {
	return dr.dr.StatusCode()
}

// Status returns the HTTP status message of the response, e.g. "200 OK".
func (dr DownloadResponse) Status() string {
	return dr.dr.Status()
}

// AcceptRanges returns the value for header Accept-Ranges.
func (dr DownloadResponse) AcceptRanges() string {
	return dr.dr.AcceptRanges()
}

// CacheControl returns the value for header Cache-Control.
func (dr DownloadResponse) CacheControl() string {
	return dr.dr.CacheControl()
}

// ContentDisposition returns the value for header Content-Disposition.
func (dr DownloadResponse) ContentDisposition() string {
	return dr.dr.ContentDisposition()
}

// ContentEncoding returns the value for header Content-Encoding.
func (dr DownloadResponse) ContentEncoding() string {
	return dr.dr.ContentEncoding()
}

// ContentLanguage returns the value for header Content-Language.
func (dr DownloadResponse) ContentLanguage() string {
	return dr.dr.ContentLanguage()
}

// ContentLength returns the value for header Content-Length.
func (dr DownloadResponse) ContentLength() int64 {
	return dr.dr.ContentLength()
}

// ContentRange returns the value for header Content-Range.
func (dr DownloadResponse) ContentRange() string {
	return dr.dr.ContentRange()
}

// ContentType returns the value for header Content-Type.
func (dr DownloadResponse) ContentType() string {
	return dr.dr.ContentType()
}

// CopyCompletionTime returns the value for header x-ms-copy-completion-time.
func (dr DownloadResponse) CopyCompletionTime() time.Time {
	return dr.dr.CopyCompletionTime()
}

// CopyID returns the value for header x-ms-copy-id.
func (dr DownloadResponse) CopyID() string {
	return dr.dr.CopyID()
}

// CopyProgress returns the value for header x-ms-copy-progress.
func (dr DownloadResponse) CopyProgress() string {
	return dr.dr.CopyProgress()
}

// CopySource returns the value for header x-ms-copy-source.
func (dr DownloadResponse) CopySource() string {
	return dr.dr.CopySource()
}

// CopyStatus returns the value for header x-ms-copy-status.
func (dr DownloadResponse) CopyStatus() CopyStatusType {
	return dr.dr.CopyStatus()
}

// CopyStatusDescription returns the value for header x-ms-copy-status-description.
func (dr DownloadResponse) CopyStatusDescription() string {
	return dr.dr.CopyStatusDescription()
}

// Date returns the value for header Date.
func (dr DownloadResponse) Date() time.Time {
	return dr.dr.Date()
}

// ETag returns the value for header ETag.
func (dr DownloadResponse) ETag() ETag {
	return dr.dr.ETag()
}

// IsServerEncrypted returns the value for header x-ms-server-encrypted.
func (dr DownloadResponse) IsServerEncrypted() string {
	return dr.dr.IsServerEncrypted()
}

// LastModified returns the value for header Last-Modified.
func (dr DownloadResponse) LastModified() time.Time {
	return dr.dr.LastModified()
}

// RequestID returns the value for header x-ms-request-id.
func (dr DownloadResponse) RequestID() string {
	return dr.dr.RequestID()
}

// Version returns the value for header x-ms-version.
func (dr DownloadResponse) Version() string {
	return dr.dr.Version()
}

// NewMetadata returns user-defined key/value pairs.
func (dr DownloadResponse) NewMetadata() Metadata {
	return dr.dr.NewMetadata()
}

// FileContentMD5 returns the value for header x-ms-content-md5.
func (dr DownloadResponse) FileContentMD5() []byte {
	return dr.dr.FileContentMD5()
}

// ContentMD5 returns the value for header Content-MD5.
func (dr DownloadResponse) ContentMD5() []byte {
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
