package sidetree

type Service interface {
	Start() error
	WaitForSync() error
	IsCurrent() bool
	BestBlock() (Block, error)
	GetBlockHash(block int) (Hash, error)
	GetBlock(hash Hash) (Block, error)
}

type Block interface {
	Height() int64
	Hash() Hash
	Transactions() []Transaction
}

type Hash interface {
	Bytes() []byte
	String() string
}

type Transaction interface {
	Hash() Hash
	TxOut() []TxOut
	Bytes() ([]byte, error)
}

type TxOut interface {
	PkScript() []byte
	Bytes() ([]byte, error)
}