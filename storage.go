package sidetree

import (
	"io"
)

type CAS interface {
	io.Closer
	Start() error
	// Will automatically unzip from gzip
	Get(id string) ([]byte, error)
	// Will automatically zip to gzip
	Put(data []byte) (string, error)
}
