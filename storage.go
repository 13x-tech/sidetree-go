package sidetree

import (
	"io"
)

type CAS interface {
	io.Closer
	Start() error
	GetGZip(id string) ([]byte, error)
	PutGZip(data []byte) (string, error)
}

type Storage interface {
	io.Closer
	CAS() (CAS, error)
}
