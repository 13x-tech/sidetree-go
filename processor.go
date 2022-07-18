package sidetree

import (
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

func Processor(op SideTreeOp, options ...SidetreeOption) (*OperationsProcessor, error) {

	if op.CID() == "" {
		return nil, fmt.Errorf("index URI is empty")
	}

	d := &OperationsProcessor{
		op:               op,
		CoreIndexFileURI: op.CID(),
	}

	for _, option := range options {
		option(d)
	}

	if d.method == "" {
		return nil, fmt.Errorf("prefix is empty")
	}

	if d.log == nil {
		return nil, fmt.Errorf("logger is not set")
	}

	if d.casStore == nil {
		return nil, fmt.Errorf("cas store is not set")
	}

	return d, nil
}

type OperationsProcessor struct {
	log    Logger
	method string
	op     SideTreeOp

	storeOps bool

	CoreIndexFileURI string
	CoreIndexFile    *CoreIndexFile

	CoreProofFileURI string
	CoreProofFile    *CoreProofFile

	ProvisionalIndexFileURI string
	ProvisionalIndexFile    *ProvisionalIndexFile

	ProvisionalProofFileURI string
	ProvisionalProofFile    *ProvisionalProofFile

	// Version 1 only has a single Chunk file No need for Array here yet
	ChunkFileURI string
	ChunkFile    *ChunkFile

	casStore CAS
	didStore DIDs

	createOps     map[string]operations.CreateInterface
	updateOps     map[string]operations.UpdateInterface
	deactivateOps map[string]operations.DeactivateInterface
	recoverOps    map[string]operations.RecoverInterface

	deltaMappingArray []string

	baseFeeFn   *BaseFeeAlgorithm
	perOpFeeFn  *PerOperationFee
	valueLockFn *ValueLocking

	baseFee int
}

type ProcessedOperations struct {
	AnchorString      string
	AnchorSequence    string
	CreateOps         map[string]operations.CreateInterface
	UpdateOps         map[string]operations.UpdateInterface
	DeactivateOps     map[string]operations.DeactivateInterface
	RecoverOps        map[string]operations.RecoverInterface
	DeltaMappingArray []string
}

func (b *OperationsProcessor) Anchor() string {
	return b.op.AnchorString
}

func (b *OperationsProcessor) SystemAnchor() string {
	return b.op.SystemAnchorPoint
}

func (d *OperationsProcessor) Process() (*ProcessedOperations, error) {

	if err := d.fetchCoreIndexFile(); err != nil {
		return nil, d.log.Errorf("core index: %s - failed to fetch core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreIndexFile == nil {
		return nil, d.log.Errorf("core index: %s - core index file is nil", d.CoreIndexFileURI)
	}

	// https://identity.foundation/sidetree/spec/#base-fee-variable
	if d.baseFeeFn != nil {
		baseFeeFn := *d.baseFeeFn
		d.baseFee = baseFeeFn(d.op.Operations(), d.op.SystemAnchorPoint)
	}

	// https://identity.foundation/sidetree/spec/#per-operation-fee
	if d.perOpFeeFn != nil {
		perOpFeeFn := *d.perOpFeeFn
		if !perOpFeeFn(d.baseFee, d.op.Operations(), d.op.SystemAnchorPoint) {
			return nil, d.log.Errorf("per op fee is not valid")
		}
	}

	// https://identity.foundation/sidetree/spec/#value-locking
	if d.valueLockFn != nil {
		valueLockFn := *d.valueLockFn
		if !valueLockFn(d.CoreIndexFile.WriterLockId, d.op.Operations(), d.baseFee, d.op.SystemAnchorPoint) {
			return nil, d.log.Errorf("value lock is not valid")
		}
	}

	if err := d.CoreIndexFile.Process(); err != nil {
		return nil, d.log.Errorf("core index: %s failed to process core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			return nil, d.log.Errorf("core index: %s - failed to fetch core proof file: %w", d.CoreIndexFileURI, err)
		}

		if d.CoreProofFile == nil {
			return nil, d.log.Errorf("core index: %s - core proof file is nil", d.CoreIndexFileURI)
		}

		if err := d.CoreProofFile.Process(); err != nil {
			return nil, d.log.Errorf("core index: %s - failed to process core proof file: %w", d.CoreIndexFileURI, err)
		}
	}

	if d.ProvisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			return nil, d.log.Errorf("core index: %s - failed to fetch provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if d.ProvisionalIndexFile == nil {
			return nil, d.log.Errorf("core index: %s - provisional index file is nil", d.CoreIndexFileURI)
		}

		if err := d.ProvisionalIndexFile.Process(); err != nil {
			return nil, d.log.Errorf("core index: %s - failed to process provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if len(d.ProvisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				return nil, d.log.Errorf("core index: %s - failed to fetch provisional proof file: %w", d.CoreIndexFileURI, err)
			}

			if d.ProvisionalProofFile == nil {
				return nil, d.log.Errorf("core index: %s - provisional proof file is nil", d.CoreIndexFileURI)
			}

			if err := d.ProvisionalProofFile.Process(); err != nil {
				return nil, d.log.Errorf("core index: %s - failed to process provisional proof file: %w", d.CoreIndexFileURI, err)
			}
		}

		if len(d.ProvisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				return nil, d.log.Errorf("core index: %s - failed to fetch chunk file: %w", d.CoreIndexFileURI, err)
			}

			if d.ChunkFile == nil {
				return nil, d.log.Errorf("core index: %s - chunk file is nil", d.CoreIndexFileURI)
			}

			if err := d.ChunkFile.Process(); err != nil {
				return nil, d.log.Errorf("core index: %s - failed to process chunk file: %w", d.CoreIndexFileURI, err)
			}
		}
	}

	return &ProcessedOperations{
		AnchorString:      d.Anchor(),
		AnchorSequence:    d.SystemAnchor(),
		CreateOps:         d.createOps,
		RecoverOps:        d.recoverOps,
		UpdateOps:         d.updateOps,
		DeactivateOps:     d.deactivateOps,
		DeltaMappingArray: d.deltaMappingArray,
	}, nil

}

func (d *OperationsProcessor) fetchCoreIndexFile() error {

	if d.CoreIndexFileURI == "" {
		return d.log.Errorf("core index file URI is empty")
	}

	coreData, err := d.casStore.GetGZip(d.CoreIndexFileURI)
	if err != nil {
		return d.log.Errorf("failed to get core index file: %w", err)
	}

	d.CoreIndexFile, err = NewCoreIndexFile(d, coreData)
	if err != nil {
		return d.log.Errorf("failed to create core index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchCoreProofFile() error {

	if d.CoreProofFileURI == "" {
		return d.log.Errorf("core proof file URI is empty")
	}

	coreProofData, err := d.casStore.GetGZip(d.CoreProofFileURI)
	if err != nil {
		return d.log.Errorf("failed to get core proof file: %w", err)
	}

	d.CoreProofFile, err = NewCoreProofFile(d, coreProofData)
	if err != nil {
		return d.log.Errorf("failed to create core proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalIndexFile() error {

	if d.ProvisionalIndexFileURI == "" {
		return d.log.Errorf("no provisional index file URI")
	}

	provisionalData, err := d.casStore.GetGZip(d.ProvisionalIndexFileURI)
	if err != nil {
		return d.log.Errorf("failed to get provisional index file: %w", err)
	}

	d.ProvisionalIndexFile, err = NewProvisionalIndexFile(d, provisionalData)
	if err != nil {
		return d.log.Errorf("failed to create provisional index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalProofFile() error {

	if d.ProvisionalProofFileURI == "" {
		return d.log.Errorf("no provisional proof file URI")
	}

	provisionalProofData, err := d.casStore.GetGZip(d.ProvisionalProofFileURI)
	if err != nil {
		return d.log.Errorf("failed to get provisional proof file: %w", err)
	}

	d.ProvisionalProofFile, err = NewProvisionalProofFile(d, provisionalProofData)
	if err != nil {
		return d.log.Errorf("failed to create provisional proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchChunkFile() error {
	if d.ChunkFileURI == "" {
		return d.log.Errorf("no chunk file URI")
	}

	chunkData, err := d.casStore.GetGZip(d.ChunkFileURI)
	if err != nil {
		return d.log.Errorf("failed to get chunk file: %w", err)
	}

	d.ChunkFile, err = NewChunkFile(d, chunkData)
	if err != nil {
		return d.log.Errorf("failed to create chunk file: %w", err)
	}

	return nil
}

func (p *OperationsProcessor) populateDeltaMappingArray() error {
	coreIndex := p.CoreIndexFile
	if coreIndex == nil {
		return fmt.Errorf("core index file is nil")
	}

	provisionalIndex := p.ProvisionalIndexFile
	if provisionalIndex == nil {
		return fmt.Errorf("provisional index file is nil")
	}

	for _, op := range coreIndex.Operations.Create {
		uri, err := op.SuffixData.URI()
		if err != nil {
			return fmt.Errorf("failed to get uri from create operation: %w", err)
		}

		createOp := operations.CreateOperation(
			p.Anchor(),
			p.SystemAnchor(),
			op.SuffixData,
		)

		p.createOps[uri] = createOp
		p.deltaMappingArray = append(p.deltaMappingArray, uri)
	}

	for _, op := range coreIndex.Operations.Recover {
		p.deltaMappingArray = append(p.deltaMappingArray, op.DIDSuffix)
	}

	for _, op := range provisionalIndex.Operations.Update {
		p.deltaMappingArray = append(p.deltaMappingArray, op.DIDSuffix)
	}

	return nil
}
