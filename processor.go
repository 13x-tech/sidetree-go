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

	if d.dids == nil {
		d.dids = []string{}
	}

	return d, nil
}

type OperationsProcessor struct {
	log    Logger
	method string
	op     SideTreeOp
	dids   []string

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
	AnchorString   string
	AnchorSequence string
	Error          error
	CreateOps      map[string]operations.CreateInterface
	UpdateOps      map[string]operations.UpdateInterface
	DeactivateOps  map[string]operations.DeactivateInterface
	RecoverOps     map[string]operations.RecoverInterface
}

func (b *OperationsProcessor) Anchor() string {
	return b.op.AnchorString
}

func (b *OperationsProcessor) SystemAnchor() string {
	return b.op.SystemAnchorPoint
}

func (d *OperationsProcessor) Process() (*ProcessedOperations, error) {

	d.createOps = map[string]operations.CreateInterface{}
	d.updateOps = map[string]operations.UpdateInterface{}
	d.deactivateOps = map[string]operations.DeactivateInterface{}
	d.recoverOps = map[string]operations.RecoverInterface{}

	ops := &ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
	}

	if err := d.fetchCoreIndexFile(); err != nil {
		//TODO Define Errors
		ops.Error = err
		return ops, d.log.Errorf("core index: %s - failed to fetch core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreIndexFile == nil {
		ops.Error = fmt.Errorf("core index file is nil")
		return ops, d.log.Errorf("core index: %s - core index file is nil", d.CoreIndexFileURI)
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
			ops.Error = fmt.Errorf("per op fee is not valid")
			return ops, d.log.Errorf("per op fee is not valid")
		}
	}

	// https://identity.foundation/sidetree/spec/#value-locking
	if d.valueLockFn != nil {
		valueLockFn := *d.valueLockFn
		if !valueLockFn(d.CoreIndexFile.WriterLockId, d.op.Operations(), d.baseFee, d.op.SystemAnchorPoint) {
			ops.Error = fmt.Errorf("value lock is not valid")
			return ops, d.log.Errorf("value lock is not valid")
		}
	}

	if err := d.CoreIndexFile.Process(); err != nil {
		ops.Error = err
		return ops, d.log.Errorf("core index: %s failed to process core index file: %w", d.CoreIndexFileURI, err)
	}

	if d.CoreProofFileURI != "" {

		if err := d.fetchCoreProofFile(); err != nil {
			ops.Error = err
			return ops, d.log.Errorf("core index: %s - failed to fetch core proof file: %w", d.CoreIndexFileURI, err)
		}

		if d.CoreProofFile == nil {
			ops.Error = fmt.Errorf("core proof file is nil")
			return ops, d.log.Errorf("core index: %s - core proof file is nil", d.CoreIndexFileURI)
		}

		if err := d.CoreProofFile.Process(); err != nil {
			ops.Error = err
			return ops, d.log.Errorf("core index: %s - failed to process core proof file: %w", d.CoreIndexFileURI, err)
		}
	}

	if d.ProvisionalIndexFileURI != "" {

		if err := d.fetchProvisionalIndexFile(); err != nil {
			ops.Error = err
			return ops, d.log.Errorf("core index: %s - failed to fetch provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if d.ProvisionalIndexFile == nil {
			ops.Error = fmt.Errorf("provisional index file is nil")
			return ops, d.log.Errorf("core index: %s - provisional index file is nil", d.CoreIndexFileURI)
		}

		if err := d.ProvisionalIndexFile.Process(); err != nil {
			ops.Error = err
			return ops, d.log.Errorf("core index: %s - failed to process provisional index file: %w", d.CoreIndexFileURI, err)
		}

		if len(d.ProvisionalIndexFile.Operations.Update) > 0 {

			if err := d.fetchProvisionalProofFile(); err != nil {
				ops.Error = err
				return ops, d.log.Errorf("core index: %s - failed to fetch provisional proof file: %w", d.CoreIndexFileURI, err)
			}

			if d.ProvisionalProofFile == nil {
				ops.Error = fmt.Errorf("provisional proof file is nil")
				return ops, d.log.Errorf("core index: %s - provisional proof file is nil", d.CoreIndexFileURI)
			}

			if err := d.ProvisionalProofFile.Process(); err != nil {
				ops.Error = err
				return ops, d.log.Errorf("core index: %s - failed to process provisional proof file: %w", d.CoreIndexFileURI, err)
			}
		}

		if len(d.ProvisionalIndexFile.Chunks) > 0 {
			if err := d.fetchChunkFile(); err != nil {
				ops.Error = err
				return ops, d.log.Errorf("core index: %s - failed to fetch chunk file: %w", d.CoreIndexFileURI, err)
			}

			if d.ChunkFile == nil {
				ops.Error = fmt.Errorf("chunk file is nil")
				return ops, d.log.Errorf("core index: %s - chunk file is nil", d.CoreIndexFileURI)
			}

			if err := d.ChunkFile.Process(); err != nil {
				ops.Error = err
				return ops, d.log.Errorf("core index: %s - failed to process chunk file: %w", d.CoreIndexFileURI, err)
			}
		}
	}

	ops = &ProcessedOperations{
		Error:          nil,
		AnchorString:   d.Anchor(),
		AnchorSequence: d.SystemAnchor(),
		CreateOps:      d.CreateOps(),
		RecoverOps:     d.RecoverOps(),
		UpdateOps:      d.UpdateOps(),
		DeactivateOps:  d.DeactivateOps(),
	}

	return ops, nil
}

func (d *OperationsProcessor) CreateOps() map[string]operations.CreateInterface {
	if len(d.dids) == 0 {
		return d.createOps
	}

	ops := map[string]operations.CreateInterface{}
	for _, did := range d.dids {
		if _, ok := d.createOps[did]; ok {
			ops[did] = d.createOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) RecoverOps() map[string]operations.RecoverInterface {
	if len(d.dids) == 0 {
		return d.recoverOps
	}

	ops := map[string]operations.RecoverInterface{}
	for _, did := range d.dids {
		if _, ok := d.recoverOps[did]; ok {
			ops[did] = d.recoverOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) UpdateOps() map[string]operations.UpdateInterface {
	if len(d.dids) == 0 {
		return d.updateOps
	}

	ops := map[string]operations.UpdateInterface{}
	for _, did := range d.dids {
		if _, ok := d.updateOps[did]; ok {
			ops[did] = d.updateOps[did]
		}
	}

	return ops
}

func (d *OperationsProcessor) DeactivateOps() map[string]operations.DeactivateInterface {
	if len(d.dids) == 0 {
		return d.deactivateOps
	}

	ops := map[string]operations.DeactivateInterface{}
	for _, did := range d.dids {
		if _, ok := d.deactivateOps[did]; ok {
			ops[did] = d.deactivateOps[did]
		}
	}

	return ops
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
		fmt.Printf("adding to delta mapping create: %s\n", uri)
		p.deltaMappingArray = append(p.deltaMappingArray, uri)
	}

	for _, op := range coreIndex.Operations.Recover {
		fmt.Printf("adding to delta mapping recover: %s\n", op.DIDSuffix)
		p.deltaMappingArray = append(p.deltaMappingArray, op.DIDSuffix)
	}

	for _, op := range provisionalIndex.Operations.Update {
		fmt.Printf("adding to delta mapping update: %s\n", op.DIDSuffix)
		p.deltaMappingArray = append(p.deltaMappingArray, op.DIDSuffix)
	}

	return nil
}
