package sidetree

import (
	"io"
)

// CASType is the type of content addressable storage.
type CASType string

func (c CASType) String() string {
	return string(c)
}

type CAS interface {
	io.Closer
	Start() error
	// Get fetches the content for id and automatically unzips it from gzip.
	//
	// maxSizeInBytes bounds the file the way a Sidetree-compliant node does
	// (mirroring the reference DownloadManager.download(uri, maxSizeInBytes)):
	// the implementation MUST refuse to read more than maxSizeInBytes of stored
	// (compressed) content and MUST bound decompression to
	// maxSizeInBytes * MaxMemoryDecompressionFactor (the zip-bomb guard). The
	// caller passes the protocol per-file cap for the file type being fetched
	// (e.g. MaxCoreIndexFileSizeInBytes). A file that exceeds the cap is
	// permanently invalid (CAS content is immutable), so the implementation
	// should return an ErrMalformed-wrapped error rather than a retryable one.
	Get(id string, maxSizeInBytes int) ([]byte, error)
	// Will automatically zip to gzip
	Put(data []byte) (string, error)
	// Type returns the type of the CAS
	Type() CASType
}
