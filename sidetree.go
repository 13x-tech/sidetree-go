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
			t.prefix = prefix
		}
	}
}

func WithStorage(storage Storage) SidetreeOption {
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

			indexStore, err := storage.Indexer()
			if err != nil {
				return
			}
			t.didStore = didStore
			t.casStore = casStore
			t.indexStore = indexStore
		}

	}
}

func WithLogger(log Logger) SidetreeOption {
	return func(d interface{}) {
		switch t := d.(type) {
		case *SideTree:
			t.log = log
		case *OperationsProcessor:
			t.log = log
		}
	}
}

func New(options ...SidetreeOption) *SideTree {
	return &SideTree{}
}

type SideTree struct {
	prefix string
	store  Storage
	log    Logger
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
