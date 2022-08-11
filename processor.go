package sidetree

import (
	"fmt"

	"github.com/13x-tech/ion-sdk-go/pkg/operations"
)

var (
	ErrInvalidMethod = fmt.Errorf("invalid method")
	ErrInvalidCAS    = fmt.Errorf("invalid cas")
	ErrEmptyURI      = fmt.Errorf("index anchor URI is empty")
)

func Processor(op operations.Anchor, options ...SideTreeOption) (*OperationsProcessor, error) {
	if op.CID() == "" {
		return nil, ErrEmptyURI
	}

	d := &OperationsProcessor{
		op:               op,
		coreIndexFileURI: op.CID(),
	}

	for _, option := range options {
		option(d)
	}

	if d.method == "" {
		return nil, ErrInvalidMethod
	}

	if d.cas == nil {
		return nil, ErrInvalidCAS
	}

	if d.filterDIDs == nil {
		d.filterDIDs = []string{}
	}

	return d, nil
}

type OperationsProcessor struct {
	cas        CAS
	filterDIDs []string
	method     string
	op         operations.Anchor

	coreIndexFileURI string
	coreIndexFile    *CoreIndexFile

	coreProofFileURI string
	coreProofFile    *CoreProofFile

	provisionalIndexFileURI string
	provisionalIndexFile    *ProvisionalIndexFile

	provisionalProofFileURI string
	provisionalProofFile    *ProvisionalProofFile

	// Version 1 only has a single Chunk file No need for Array here yet
	chunkFileURI string
	chunkFile    *ChunkFile

	createOps     map[string]operations.CreateInterface
	updateOps     map[string]operations.UpdateInterface
	deactivateOps map[string]operations.DeactivateInterface
	recoverOps    map[string]operations.RecoverInterface

	createMappingArray   []string
	recoveryMappingArray []string
	updateMappingArray   []string

	//TODO These Don't actually do Anything Yet
	baseFeeFn   BaseFeeAlgorithm
	perOpFeeFn  PerOperationFee
	valueLockFn ValueLocking

	baseFee int
}

type ProcessedOperations struct {
	AnchorString   string
	AnchorSequence string
	Error          error
	CreateOps      map[string]operations.CreateInterface
	UpdateOps      map[string]operations.UpdateInterface
	DeactivateOps  map[string]operations.DeactivateInterface
	RecoverOps     map[string]operations.RecoverInterface
}

func (b *OperationsProcessor) Anchor() string {
	return string(b.op.Anchor)
}

func (b *OperationsProcessor) SystemAnchor() string {
	return string(b.op.Sequence)
}

func (d *OperationsProcessor) Process() ProcessedOperations {

	d.createMappingArray = []string{}
	d.recoveryMappingArray = []string{}
	d.updateMappingArray = []string{}

	d.createOps = map[string]operations.CreateInterface{}
	d.updateOps = map[string]operations.UpdateInterface{}
	d.deactivateOps = map[string]operations.DeactivateInterface{}
	d.recoverOps = map[string]operations.RecoverInterface{}

	ops := ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
	}

	if err := d.fetchCoreIndexFile(); err != nil {
		//TODO Define Errors
		ops.Error = err
		return ops
	}

	// https://identity.foundation/sidetree/spec/#base-fee-variable
	if d.baseFeeFn != nil {
		d.baseFee = d.baseFeeFn(d.op.Operations(), string(d.op.Sequence))
	}

	// https://identity.foundation/sidetree/spec/#per-operation-fee
	if d.perOpFeeFn != nil {
		if !d.perOpFeeFn(d.baseFee, d.op.Operations(), string(d.op.Sequence)) {
			ops.Error = fmt.Errorf("per op fee is not valid")
			return ops
		}
	}

	// https://identity.foundation/sidetree/spec/#value-locking
	if d.valueLockFn != nil {
		if !d.valueLockFn(d.coreIndexFile.WriterLockId, d.op.Operations(), d.baseFee, string(d.op.Sequence)) {
			ops.Error = fmt.Errorf("value lock is not valid")
			return ops
		}
	}

	if err := d.coreIndexFile.Process(); err != nil {
		ops.Error = err
		return ops
	}

	if d.coreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.coreProofFile.Process(); err != nil {
			ops.Error = err
			return ops
		}
	}

	if d.provisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			ops.Error = err
			return ops
		}

		if err := d.provisionalIndexFile.Process(); err != nil {
			ops.Error = err
			return ops
		}

		if len(d.provisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.provisionalProofFile.Process(); err != nil {
				ops.Error = err
				return ops
			}
		}

		if len(d.provisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				ops.Error = err
				return ops
			}

			if err := d.chunkFile.Process(); err != nil {
				ops.Error = err
				return ops
			}
		}
	}

	return ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
		CreateOps:      d.CreateOps(),
		RecoverOps:     d.RecoverOps(),
		UpdateOps:      d.UpdateOps(),
		DeactivateOps:  d.DeactivateOps(),
	}
}

func (d *OperationsProcessor) CreateOps() map[string]operations.CreateInterface {
	if len(d.filterDIDs) == 0 {
		return d.createOps
	}

	ops := map[string]operations.CreateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.createOps[did]; ok {
			ops[did] = d.createOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) RecoverOps() map[string]operations.RecoverInterface {
	if len(d.filterDIDs) == 0 {
		return d.recoverOps
	}

	ops := map[string]operations.RecoverInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.recoverOps[did]; ok {
			ops[did] = d.recoverOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) UpdateOps() map[string]operations.UpdateInterface {
	if len(d.filterDIDs) == 0 {
		return d.updateOps
	}

	ops := map[string]operations.UpdateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.updateOps[did]; ok {
			ops[did] = d.updateOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) DeactivateOps() map[string]operations.DeactivateInterface {
	if len(d.filterDIDs) == 0 {
		return d.deactivateOps
	}

	ops := map[string]operations.DeactivateInterface{}
	for _, did := range d.filterDIDs {
		if _, ok := d.deactivateOps[did]; ok {
			ops[did] = d.deactivateOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) fetchCoreIndexFile() error {

	coreData, err := d.cas.Get(d.coreIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core index file: %w", err)
	}

	d.coreIndexFile, err = NewCoreIndexFile(d, coreData)
	if err != nil {
		return fmt.Errorf("failed to create core index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchCoreProofFile() error {

	coreProofData, err := d.cas.Get(d.coreProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get core proof file: %w", err)
	}

	d.coreProofFile, err = NewCoreProofFile(d, coreProofData)
	if err != nil {
		return fmt.Errorf("failed to create core proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalIndexFile() error {

	provisionalData, err := d.cas.Get(d.provisionalIndexFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional index file: %w", err)
	}

	d.provisionalIndexFile, err = NewProvisionalIndexFile(d, provisionalData)
	if err != nil {
		return fmt.Errorf("failed to create provisional index file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchProvisionalProofFile() error {

	provisionalProofData, err := d.cas.Get(d.provisionalProofFileURI)
	if err != nil {
		return fmt.Errorf("failed to get provisional proof file: %w", err)
	}

	d.provisionalProofFile, err = NewProvisionalProofFile(d, provisionalProofData)
	if err != nil {
		return fmt.Errorf("failed to create provisional proof file: %w", err)
	}

	return nil
}

func (d *OperationsProcessor) fetchChunkFile() error {

	chunkData, err := d.cas.Get(d.chunkFileURI)
	if err != nil {
		return fmt.Errorf("failed to get chunk file: %w", err)
	}

	d.chunkFile, err = NewChunkFile(chunkData,
		WithMappingArrays(d.createMappingArray, d.recoveryMappingArray, d.updateMappingArray),
		WithOperations(d.createOps, d.recoverOps, d.updateOps),
	)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}

	return nil
}

func (p *OperationsProcessor) populateDeltaMappingArray() error {

	provisionalIndex := p.provisionalIndexFile
	if provisionalIndex == nil {
		return fmt.Errorf("provisional index file is nil")
	}

	for _, op := range p.coreIndexFile.Operations.Create {
		uri, _ := op.SuffixData.URI()

		createOp := operations.CreateOperation(
			op.SuffixData,
		)

		p.createOps[uri] = createOp
		p.createMappingArray = append(p.createMappingArray, uri)
	}

	for _, op := range p.coreIndexFile.Operations.Recover {
		p.recoveryMappingArray = append(p.recoveryMappingArray, op.DIDSuffix)
	}

	for _, op := range provisionalIndex.Operations.Update {
		p.updateMappingArray = append(p.updateMappingArray, op.DIDSuffix)
	}

	return nil
}
