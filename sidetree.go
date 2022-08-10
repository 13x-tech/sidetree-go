package sidetree

import (
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

type SideTreeOption func(interface{})

func WithDIDs(filteredDIDs []string) SideTreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			return
		case *OperationsProcessor:
			t.filterDIDs = filteredDIDs
		}
	}
}

func WithPrefix(prefix string) SideTreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.method = prefix
		case *OperationsProcessor:
			t.method = prefix
		}
	}
}

func WithCAS(cas CAS) SideTreeOption {
	//TODO return error?
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.cas = cas
		case *OperationsProcessor:
			t.cas = cas
		}
	}
}

func WithFeeFunctions(feeFunctions ...interface{}) SideTreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			for _, f := range feeFunctions {
				switch fn := f.(type) {
				case BaseFeeAlgorithm:
					t.baseFeeFn = fn
				case PerOperationFee:
					t.perOpFeeFn = fn
				case ValueLocking:
					t.valueLockFn = fn
				}
			}
		case *OperationsProcessor:
			for _, f := range feeFunctions {
				switch fn := f.(type) {
				case BaseFeeAlgorithm:
					t.baseFeeFn = fn
				case PerOperationFee:
					t.perOpFeeFn = fn
				case ValueLocking:
					t.valueLockFn = fn
				}
			}
		}
	}
}

func New(options ...SideTreeOption) *SideTree {
	s := &SideTree{}
	for _, option := range options {
		option(s)
	}
	return s
}

//TODO have better defined variables that could fit multiple anchoring systems
type BaseFeeAlgorithm func(opCount int, anchorPoint string) int
type PerOperationFee func(baseFee int, opCount int, anchorPoint string) bool
type ValueLocking func(writerLockId string, baseFee int, opCount int, anchorPoint string) bool

type SideTree struct {
	method      string
	cas         CAS
	baseFeeFn   BaseFeeAlgorithm
	perOpFeeFn  PerOperationFee
	valueLockFn ValueLocking
}

func (s *SideTree) ProcessOperations(ops []operations.Anchor, ids []string) (map[operations.Anchor]ProcessedOperations, error) {

	//TODO Validate ids

	opsMap := map[operations.Anchor]ProcessedOperations{}
	for _, op := range ops {

		processor, err := Processor(
			op,
			WithPrefix(s.method),
			WithCAS(s.cas),
			WithDIDs(ids),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create operations processor: %w", err)
		}

		opsMap[op] = processor.Process()
	}

	return opsMap, nil
}
