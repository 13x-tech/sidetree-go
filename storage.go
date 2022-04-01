package sidetree

import (
	"io"

	"github.com/13x-tech/sidetree-go/internal/did"
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
	Put(doc *did.Doc) error
	Deactivate(id string) error
	Recover(id string) error
	Get(id string) (*did.Doc, error)
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

type WalletStore interface {
	io.Closer
	PutRecoveryKey(id string, key []byte) error
	GetRecoveryKey(id string) ([]byte, error)
	PutUpdateKey(id string, key []byte) error
	GetUpdateKey(id string) ([]byte, error)
	PutUpdateReveal(id string, reveal string) error
	GetUpdateReveal(id string) (string, error)
	PutRecoveryReveal(id string, reveal string) error
	GetRecoveryReveal(id string) (string, error)
}
