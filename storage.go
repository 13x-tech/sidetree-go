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
	// Will automatically unzip from gzip
	Get(id string) ([]byte, error)
	// Will automatically zip to gzip
	Put(data []byte) (string, error)
	// Type returns the type of the CAS
	Type() CASType
}
