package sidetree

import (
	"fmt"
)

type SidetreeOption func(interface{})

func WithPrefix(prefix string) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.prefix = prefix
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
			didStore, err := storage.DIDs()
			if err != nil {
				return
			}

			casStore, err := storage.CAS()
			if err != nil {
				return
			}

			t.didStore = didStore
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
	prefix      string
	store       Storage
	log         Logger
	baseFeeFn   *BaseFeeAlgorithm
	perOpFeeFn  *PerOperationFee
	valueLockFn *ValueLocking
}

func (s *SideTree) ProcessOperations(ops []SideTreeOp) error {

	for _, op := range ops {
		processor, err := Processor(
			op,
			WithPrefix(s.prefix),
			WithStorage(s.store),
			WithLogger(s.log),
		)
		if err != nil {
			return fmt.Errorf("failed to create operations processor: %w", err)
		}

		if err := processor.Process(); err != nil {
			return fmt.Errorf("failed to process operations: %w", err)
		}
	}

	return nil
}
