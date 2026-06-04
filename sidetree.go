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

// TODO have better defined variables that could fit multiple anchoring systems
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

// feeFunctions returns the configured fee / value-lock callbacks as a slice
// suitable for WithFeeFunctions, omitting any that are unset. A nil callback is
// never forwarded so the nil guards in OperationsProcessor.Process() stay correct.
func (s *SideTree) feeFunctions() []interface{} {
	var fns []interface{}
	if s.baseFeeFn != nil {
		fns = append(fns, s.baseFeeFn)
	}
	if s.perOpFeeFn != nil {
		fns = append(fns, s.perOpFeeFn)
	}
	if s.valueLockFn != nil {
		fns = append(fns, s.valueLockFn)
	}
	return fns
}

func (s *SideTree) ProcessOperations(ops []operations.Anchor, ids []string) (map[operations.Anchor]ProcessedOperations, error) {

	//TODO Validate ids

	// Forward any configured fee / value-lock callbacks to every per-anchor
	// Processor. Without this the base-fee, per-operation-fee, and value-lock
	// checks in Process() can never fire (the Processor's callbacks would stay
	// nil), so a SideTree built WithFeeFunctions would silently skip them.
	// Only non-nil callbacks are forwarded, keeping the nil guards in Process()
	// correct for a SideTree configured with a subset of the callbacks.
	feeFns := s.feeFunctions()

	opsMap := map[operations.Anchor]ProcessedOperations{}
	for _, op := range ops {

		opts := []SideTreeOption{
			WithPrefix(s.method),
			WithCAS(s.cas),
			WithDIDs(ids),
		}
		if len(feeFns) > 0 {
			opts = append(opts, WithFeeFunctions(feeFns...))
		}

		processor, err := Processor(op, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create operations processor: %w", err)
		}

		opsMap[op] = processor.Process()
	}

	return opsMap, nil
}
