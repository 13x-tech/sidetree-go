package sidetree

import (
	"fmt"
)

type SidetreeOption func(interface{})

func WithDIDs(dids []string) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
		case *OperationsProcessor:
			t.dids = dids
		}
	}
}

func WithPrefix(prefix string) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.method = prefix
		case *OperationsProcessor:
			t.method = prefix
		}
	}
}

func WithStorage(storage Storage) SidetreeOption {
	if storage == nil {
		panic("storage is nil")
	}

	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.store = storage
		case *OperationsProcessor:
			casStore, err := storage.CAS()
			if err != nil {
				return
			}

			t.casStore = casStore
		}

	}
}

func WithLogger(log Logger) SidetreeOption {
	if log == nil {
		panic("log is nil")
	}

	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.log = log
		case *OperationsProcessor:
			t.log = log
		}
	}
}

func WithFeeFunctions(feeFunctions ...interface{}) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			for _, f := range feeFunctions {
				switch fn := f.(type) {
				case BaseFeeAlgorithm:
					t.baseFeeFn = &fn
				case PerOperationFee:
					t.perOpFeeFn = &fn
				case ValueLocking:
					t.valueLockFn = &fn
				}
			}
		case *OperationsProcessor:
			for _, f := range feeFunctions {
				switch fn := f.(type) {
				case BaseFeeAlgorithm:
					t.baseFeeFn = &fn
				case PerOperationFee:
					t.perOpFeeFn = &fn
				case ValueLocking:
					t.valueLockFn = &fn
				}
			}
		}
	}
}

func New(options ...SidetreeOption) *SideTree {
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
	store       Storage
	log         Logger
	baseFeeFn   *BaseFeeAlgorithm
	perOpFeeFn  *PerOperationFee
	valueLockFn *ValueLocking
}

func (s *SideTree) ProcessOperations(ops []SideTreeOp, ids []string) (map[SideTreeOp]ProcessedOperations, error) {

	//TODO Validate ids

	opsMap := map[SideTreeOp]ProcessedOperations{}
	for _, op := range ops {
		processor, err := Processor(
			op,
			WithPrefix(s.method),
			WithStorage(s.store),
			WithLogger(s.log),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create operations processor: %w", err)
		}

		processed, err := processor.Process()
		if err != nil {
			s.log.Errorf("failed to process operation: %s", err)
			// return nil, fmt.Errorf("failed to process operations: %w", err)
		}
		opsMap[op] = *processed
	}

	return opsMap, nil
}
