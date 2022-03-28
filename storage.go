package sidetree

import (
	"io"
)

type CAS interface {
	Start() error
	GetGZip(id string) ([]byte, error)
}

type Storage interface {
	io.Closer
	CAS() (CAS, error)
	DIDs() (DIDs, error)
	Indexer() (Indexer, error)
}

type DIDs interface {
	io.Closer
	Put(doc *DIDDoc) error
	Deactivate(id string) error
	Recover(id string) error
	Get(id string) (*DIDDoc, error)
	List() ([]string, error)
}

// TODO: Refactor this to be generalized amongst different anchoring systems
// Currently best suited for ION
type Indexer interface {
	io.Closer
	PutOps(index int, ops []SideTreeOp) error
	GetOps(index int) ([]SideTreeOp, error)
	PutDIDOps(id string, ops []SideTreeOp) error
	GetDIDOps(id string) ([]SideTreeOp, error)
}
