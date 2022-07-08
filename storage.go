package sidetree

import (
	"io"

	"github.com/13x-tech/ion-sdk-go/pkg/did"
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
	DIDs() (DIDs, error)
	Indexer() (Indexer, error)
}

type DIDs interface {
	io.Closer
	Put(doc *did.Document) error
	Deactivate(id string) error
	Recover(id string) error
	Get(id string) (*did.Document, error)
}

// TODO: Refactor this to be generalized amongst different anchoring systems
// Currently best suited for ION
type Indexer interface {
	io.Closer
	PutOps(index int, ops []SideTreeOp) error
	GetOps(index int) ([]SideTreeOp, error)
	IsProcessed(height int64) (bool, error)
	SetProcessed(height int64, hash string) error
	LastSynced() (int, error)
}
