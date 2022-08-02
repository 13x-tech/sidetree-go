package sidetree

import "fmt"

var (
	ErrURINotFound        = fmt.Errorf("URI not found")
	ErrDuplicateOperation = fmt.Errorf("duplicate operation")
	ErrNoCoreProof        = fmt.Errorf("core proof uri is empty")
)
