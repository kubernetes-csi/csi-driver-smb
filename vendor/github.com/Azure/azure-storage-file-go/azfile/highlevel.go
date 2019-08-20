package azfile

import (
	"context"
	"errors"
	"fmt"
	"io"

	"bytes"
	"os"
	"sync"

	"github.com/Azure/azure-pipeline-go/pipeline"
)

const (
	// defaultParallelCount specifies default parallel count will be used by parallel upload/download methods
	defaultParallelCount = 5

	// fileSegmentSize specifies file segment size that file would be splitted into during parallel upload/download
	fileSegmentSize = 500 * 1024 * 1024
)

// UploadToAzureFileOptions identifies options used by the UploadBufferToAzureFile and UploadFileToAzureFile functions.
type UploadToAzureFileOptions struct {
	// RangeSize specifies the range size to use in each parallel upload; the default (and maximum size) is FileMaxUploadRangeBytes.
	RangeSize int64

	// Progress is a function that is invoked periodically as bytes are send in a UploadRange call to the FileURL.
	Progress pipeline.ProgressReceiver

	// Parallelism indicates the maximum number of ranges to upload in parallel. If 0(default) is provided, 5 parallelism will be used by default.
	Parallelism uint16

	// FileHTTPHeaders contains read/writeable file properties.
	FileHTTPHeaders FileHTTPHeaders

	// Metadata contains metadata key/value pairs.
	Metadata Metadata
}

// UploadBufferToAzureFile uploads a buffer to an Azure file.
// Note: o.RangeSize must be >= 0 and <= FileMaxUploadRangeBytes, and if not specified, method will use FileMaxUploadRangeBytes by default.
// The total size to be uploaded should be <= FileMaxSizeInBytes.
func UploadBufferToAzureFile(ctx context.Context, b []byte,
	fileURL FileURL, o UploadToAzureFileOptions) error {

	// 1. Validate parameters, and set defaults.
	if o.RangeSize < 0 || o.RangeSize > FileMaxUploadRangeBytes {
		return fmt.Errorf("invalid argument, o.RangeSize must be >= 0 and <= %d, in bytes", FileMaxUploadRangeBytes)
	}
	if o.RangeSize == 0 {
		o.RangeSize = FileMaxUploadRangeBytes
	}

	size := int64(len(b))

	parallelism := o.Parallelism
	if parallelism == 0 {
		parallelism = defaultParallelCount // default parallelism
	}

	// 2. Try to create the Azure file.
	_, err := fileURL.Create(ctx, size, o.FileHTTPHeaders, o.Metadata)
	if err != nil {
		return err
	}
	// If size equals to 0, upload nothing and directly return.
	if size == 0 {
		return nil
	}

	// 3. Prepare and do parallel upload.
	fileProgress := int64(0)
	progressLock := &sync.Mutex{}

	return doBatchTransfer(ctx, batchTransferOptions{
		transferSize: size,
		chunkSize:    o.RangeSize,
		parallelism:  parallelism,
		operation: func(offset int64, curRangeSize int64) error {
			// Prepare to read the proper section of the buffer.
			var body io.ReadSeeker = bytes.NewReader(b[offset : offset+curRangeSize])
			if o.Progress != nil {
				rangeProgress := int64(0)
				body = pipeline.NewRequestBodyProgress(body,
					func(bytesTransferred int64) {
						diff := bytesTransferred - rangeProgress
						rangeProgress = bytesTransferred
						progressLock.Lock()
						defer progressLock.Unlock()
						fileProgress += diff
						o.Progress(fileProgress)
					})
			}

			_, err := fileURL.UploadRange(ctx, int64(offset), body, nil)
			return err
		},
		operationName: "UploadBufferToAzureFile",
	})
}

// UploadFileToAzureFile uploads a local file to an Azure file.
func UploadFileToAzureFile(ctx context.Context, file *os.File,
	fileURL FileURL, o UploadToAzureFileOptions) error {

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	m := mmf{} // Default to an empty slice; used for 0-size file
	if stat.Size() != 0 {
		m, err = newMMF(file, false, 0, int(stat.Size()))
		if err != nil {
			return err
		}
		defer m.unmap()
	}
	return UploadBufferToAzureFile(ctx, m, fileURL, o)
}

// DownloadFromAzureFileOptions identifies options used by the DownloadAzureFileToBuffer and DownloadAzureFileToFile functions.
type DownloadFromAzureFileOptions struct {
	// RangeSize specifies the range size to use in each parallel download; the default is FileMaxUploadRangeBytes.
	RangeSize int64

	// Progress is a function that is invoked periodically as bytes are recieved.
	Progress pipeline.ProgressReceiver

	// Parallelism indicates the maximum number of ranges to download in parallel. If 0(default) is provided, 5 parallelism will be used by default.
	Parallelism uint16

	// Max retry requests used during reading data for each range.
	MaxRetryRequestsPerRange int
}

// downloadAzureFileToBuffer downloads an Azure file to a buffer with parallel.
// Note: o.RangeSize must be >= 0.
func downloadAzureFileToBuffer(ctx context.Context, fileURL FileURL, azfileProperties *FileGetPropertiesResponse,
	b []byte, o DownloadFromAzureFileOptions) (*FileGetPropertiesResponse, error) {

	// 1. Validate parameters, and set defaults.
	if o.RangeSize < 0 {
		return nil, errors.New("invalid argument, o.RangeSize must be >= 0")
	}
	if o.RangeSize == 0 {
		o.RangeSize = FileMaxUploadRangeBytes
	}

	if azfileProperties == nil {
		p, err := fileURL.GetProperties(ctx)
		if err != nil {
			return nil, err
		}
		azfileProperties = p
	}
	azfileSize := azfileProperties.ContentLength()

	// If azure file size equals to 0, directly return as nothing need be downloaded.
	if azfileSize == 0 {
		return azfileProperties, nil
	}

	if int64(len(b)) < azfileSize {
		sanityCheckFailed(fmt.Sprintf("The buffer's size should be equal to or larger than Azure file's size: %d.", azfileSize))
	}

	parallelism := o.Parallelism
	if parallelism == 0 {
		parallelism = defaultParallelCount // default parallelism
	}

	// 2. Prepare and do parallel download.
	fileProgress := int64(0)
	progressLock := &sync.Mutex{}

	err := doBatchTransfer(ctx, batchTransferOptions{
		transferSize: azfileSize,
		chunkSize:    o.RangeSize,
		parallelism:  parallelism,
		operation: func(offset int64, curRangeSize int64) error {
			dr, err := fileURL.Download(ctx, offset, curRangeSize, false)
			body := dr.Body(RetryReaderOptions{MaxRetryRequests: o.MaxRetryRequestsPerRange})

			if o.Progress != nil {
				rangeProgress := int64(0)
				body = pipeline.NewResponseBodyProgress(
					body,
					func(bytesTransferred int64) {
						diff := bytesTransferred - rangeProgress
						rangeProgress = bytesTransferred
						progressLock.Lock()
						defer progressLock.Unlock()
						fileProgress += diff
						o.Progress(fileProgress)
					})
			}

			_, err = io.ReadFull(body, b[offset:offset+curRangeSize])
			body.Close()

			return err
		},
		operationName: "downloadAzureFileToBuffer",
	})
	if err != nil {
		return nil, err
	}

	return azfileProperties, nil
}

// DownloadAzureFileToBuffer downloads an Azure file to a buffer with parallel.
func DownloadAzureFileToBuffer(ctx context.Context, fileURL FileURL,
	b []byte, o DownloadFromAzureFileOptions) (*FileGetPropertiesResponse, error) {
	return downloadAzureFileToBuffer(ctx, fileURL, nil, b, o)
}

// DownloadAzureFileToFile downloads an Azure file to a local file.
// The file would be created if it doesn't exist, and would be truncated if the size doesn't match.
// Note: file can't be nil.
func DownloadAzureFileToFile(ctx context.Context, fileURL FileURL, file *os.File, o DownloadFromAzureFileOptions) (*FileGetPropertiesResponse, error) {
	// 1. Validate parameters.
	if file == nil {
		return nil, errors.New("invalid argument, file can't be nil")
	}

	// 2. Try to get Azure file's size.
	azfileProperties, err := fileURL.GetProperties(ctx)
	if err != nil {
		return nil, err
	}
	azfileSize := azfileProperties.ContentLength()

	// 3. Compare and try to resize local file's size if it doesn't match Azure file's size.
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() != azfileSize {
		if err = file.Truncate(azfileSize); err != nil {
			return nil, err
		}
	}

	// 4. Set mmap and call DownloadAzureFileToBuffer, in this case file size should be > 0.
	m := mmf{} // Default to an empty slice; used for 0-size file
	if azfileSize > 0 {
		m, err = newMMF(file, true, 0, int(azfileSize))
		if err != nil {
			return nil, err
		}
		defer m.unmap()
	}

	return downloadAzureFileToBuffer(ctx, fileURL, azfileProperties, m, o)
}

// BatchTransferOptions identifies options used by doBatchTransfer.
type batchTransferOptions struct {
	transferSize  int64
	chunkSize     int64
	parallelism   uint16
	operation     func(offset int64, chunkSize int64) error
	operationName string
}

// doBatchTransfer helps to execute operations in a batch manner.
func doBatchTransfer(ctx context.Context, o batchTransferOptions) error {
	// Prepare and do parallel operations.
	numChunks := ((o.transferSize - 1) / o.chunkSize) + 1
	operationChannel := make(chan func() error, o.parallelism) // Create the channel that release 'parallelism' goroutines concurrently
	operationResponseChannel := make(chan error, numChunks)    // Holds each response
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create the goroutines that process each operation (in parallel).
	for g := uint16(0); g < o.parallelism; g++ {
		//grIndex := g
		go func() {
			for f := range operationChannel {
				//fmt.Printf("[%s] gr-%d start action\n", o.operationName, grIndex)
				err := f()
				operationResponseChannel <- err
				//fmt.Printf("[%s] gr-%d end action\n", o.operationName, grIndex)
			}
		}()
	}

	curChunkSize := o.chunkSize
	// Add each chunk's operation to the channel.
	for chunkIndex := int64(0); chunkIndex < numChunks; chunkIndex++ {
		if chunkIndex == numChunks-1 { // Last chunk
			curChunkSize = o.transferSize - (int64(chunkIndex) * o.chunkSize) // Remove size of all transferred chunks from total
		}
		offset := int64(chunkIndex) * o.chunkSize

		closureChunkSize := curChunkSize
		operationChannel <- func() error {
			return o.operation(offset, closureChunkSize)
		}
	}
	close(operationChannel)

	// Wait for the operations to complete.
	for chunkIndex := int64(0); chunkIndex < numChunks; chunkIndex++ {
		responseError := <-operationResponseChannel
		if responseError != nil {
			cancel()             // As soon as any operation fails, cancel all remaining operation calls
			return responseError // No need to process anymore responses
		}
	}
	return nil
}
