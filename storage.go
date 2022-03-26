package sidetree

import "io"

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

type Indexer interface {
	io.Closer
	PutBlockOps(height int64, ops []SideTreeOp) error
	GetBlockOps(height int64) ([]SideTreeOp, error)
	PutUnreachableCID(cid string) error
	ListUnreachableCIDS() ([]string, error)
}

type SideTreeOp struct {
	BlockHash    string
	Height       int64
	BlockTxIndex int
	TxOutpoint   string
	Ops          int
	CID          string
	Processed    bool
}
