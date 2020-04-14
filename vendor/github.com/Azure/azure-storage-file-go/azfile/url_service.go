package azfile

import (
	"context"
	"net/url"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

const (
	// storageAnalyticsVersion indicates the version of Storage Analytics to configure. Use "1.0" for this value.
	// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/set-file-service-properties.
	storageAnalyticsVersion = "1.0"
)

// A ServiceURL represents a URL to the Azure Storage File service allowing you to manipulate file shares.
type ServiceURL struct {
	client serviceClient
}

// NewServiceURL creates a ServiceURL object using the specified URL and request policy pipeline.
// Note: p can't be nil.
func NewServiceURL(url url.URL, p pipeline.Pipeline) ServiceURL {
	client := newServiceClient(url, p)
	return ServiceURL{client: client}
}

// URL returns the URL endpoint used by the ServiceURL object.
func (s ServiceURL) URL() url.URL {
	return s.client.URL()
}

// String returns the URL as a string.
func (s ServiceURL) String() string {
	u := s.URL()
	return u.String()
}

// WithPipeline creates a new ServiceURL object identical to the source but with the specified request policy pipeline.
func (s ServiceURL) WithPipeline(p pipeline.Pipeline) ServiceURL {
	return NewServiceURL(s.URL(), p)
}

// NewShareURL creates a new ShareURL object by concatenating shareName to the end of
// ServiceURL's URL. The new ShareURL uses the same request policy pipeline as the ServiceURL.
// To change the pipeline, create the ShareURL and then call its WithPipeline method passing in the
// desired pipeline object. Or, call this package's NewShareURL instead of calling this object's
// NewShareURL method.
func (s ServiceURL) NewShareURL(shareName string) ShareURL {
	shareURL := appendToURLPath(s.URL(), shareName)
	return NewShareURL(shareURL, s.client.Pipeline())
}

// appendToURLPath appends a string to the end of a URL's path (prefixing the string with a '/' if required)
func appendToURLPath(u url.URL, name string) url.URL {
	// e.g. "https://ms.com/a/b/?k1=v1&k2=v2#f"
	// When you call url.Parse() this is what you'll get:
	//     Scheme: "https"
	//     Opaque: ""
	//       User: nil
	//       Host: "ms.com"
	//       Path: "/a/b/"	This should start with a / and it might or might not have a trailing slash
	//    RawPath: ""
	// ForceQuery: false
	//   RawQuery: "k1=v1&k2=v2"
	//   Fragment: "f"
	if len(u.Path) == 0 || u.Path[len(u.Path)-1] != '/' {
		u.Path += "/" // Append "/" to end before appending name
	}
	u.Path += name
	return u
}

// ListSharesSegment returns a single segment of shares starting from the specified Marker. Use an empty
// Marker to start enumeration from the beginning. Share names are returned in lexicographic order.
// After getting a segment, process it, and then call ListSharesSegment again (passing the the previously-returned
// Marker) to get the next segment. For more information, see
// https://docs.microsoft.com/en-us/rest/api/storageservices/list-shares.
func (s ServiceURL) ListSharesSegment(ctx context.Context, marker Marker, o ListSharesOptions) (*ListSharesResponse, error) {
	prefix, include, maxResults := o.pointers()
	return s.client.ListSharesSegment(ctx, prefix, marker.Val, maxResults, include, nil)
}

// ListSharesOptions defines options available when calling ListSharesSegment.
type ListSharesOptions struct {
	Detail     ListSharesDetail // No IncludeType header is produced if ""
	Prefix     string           // No Prefix header is produced if ""
	MaxResults int32            // 0 means unspecified
}

func (o *ListSharesOptions) pointers() (prefix *string, include []ListSharesIncludeType, maxResults *int32) {
	if o.Prefix != "" {
		prefix = &o.Prefix
	}
	if o.MaxResults != 0 {
		maxResults = &o.MaxResults
	}
	include = o.Detail.toArray()
	return
}

// ListSharesDetail indicates what additional information the service should return with each share.
type ListSharesDetail struct {
	Metadata, Snapshots bool
}

// toArray produces the Include query parameter's value.
func (d *ListSharesDetail) toArray() []ListSharesIncludeType {
	items := make([]ListSharesIncludeType, 0, 2)
	if d.Metadata {
		items = append(items, ListSharesIncludeMetadata)
	}
	if d.Snapshots {
		items = append(items, ListSharesIncludeSnapshots)
	}

	return items
}

// toFsp converts StorageServiceProperties to convenience representation FileServiceProperties.
// This method is added considering protocol layer's swagger unification purpose.
func (ssp *StorageServiceProperties) toFsp() *FileServiceProperties {
	if ssp == nil {
		return nil
	}

	return &FileServiceProperties{
		rawResponse:   ssp.rawResponse,
		HourMetrics:   ssp.HourMetrics.toMp(),
		MinuteMetrics: ssp.MinuteMetrics.toMp(),
		Cors:          ssp.Cors,
	}
}

// toMp converts Metrics to convenience representation MetricProperties.
// This method is added considering protocol layer's swagger unification purpose.
func (m *Metrics) toMp() MetricProperties {
	mp := MetricProperties{}
	if m.Enabled {
		mp.MetricEnabled = true
		mp.IncludeAPIs = *m.IncludeAPIs
		if m.RetentionPolicy != nil && m.RetentionPolicy.Enabled {
			mp.RetentionPolicyEnabled = true
			mp.RetentionDays = *m.RetentionPolicy.Days
		}
	}

	return mp
}

// toSsp converts FileServiceProperties to convenience representation StorageServiceProperties.
// This method is added considering protocol layer's swagger unification purpose.
func (fsp *FileServiceProperties) toSsp() *StorageServiceProperties {
	if fsp == nil {
		return nil
	}

	return &StorageServiceProperties{
		rawResponse:   fsp.rawResponse,
		HourMetrics:   fsp.HourMetrics.toM(),
		MinuteMetrics: fsp.MinuteMetrics.toM(),
		Cors:          fsp.Cors,
	}
}

// toM converts MetricProperties to Metrics.
// This method is added considering protocol layer's swagger unification purpose.
func (mp MetricProperties) toM() *Metrics {
	m := Metrics{
		Version:         storageAnalyticsVersion,
		RetentionPolicy: &RetentionPolicy{}} // Note: Version and RetentionPolicy are actually mandatory.

	if mp.MetricEnabled {
		m.Enabled = true
		m.IncludeAPIs = &mp.IncludeAPIs
		if mp.RetentionPolicyEnabled {
			m.RetentionPolicy.Enabled = true
			m.RetentionPolicy.Days = &mp.RetentionDays
		}
	}

	return &m
}

// GetProperties returns the properties of the File service.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/get-file-service-properties.
func (s ServiceURL) GetProperties(ctx context.Context) (*FileServiceProperties, error) {
	ssp, error := s.client.GetProperties(ctx, nil)

	return ssp.toFsp(), error
}

// SetProperties sets the properties of the File service.
// For more information, see https://docs.microsoft.com/en-us/rest/api/storageservices/set-file-service-properties.
func (s ServiceURL) SetProperties(ctx context.Context, properties FileServiceProperties) (*ServiceSetPropertiesResponse, error) {
	return s.client.SetProperties(ctx, *properties.toSsp(), nil)
}
